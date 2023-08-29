/*
 *    Copyright 2023 iFood
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package scan

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"errors"
	"fmt"
	"github.com/hillu/go-yara/v4"
	"io"
	"os"
	"path/filepath"
)

type YaraScanner struct {
	scanner *yara.Scanner
	logger  logging.Logger
}

func NewYaraScanner(rulesDir string, logger logging.Logger) (*YaraScanner, error) {
	yaraScanner := &YaraScanner{logger: logger}

	if rulesDir == "" {
		logger.Infow("No Yara rules dir was specified, proceeding without yara scan.")
		return yaraScanner, nil
	}
	scanner, err := yaraScanner.createScanner(rulesDir)
	yaraScanner.scanner = scanner

	return yaraScanner, err
}

func (y *YaraScanner) createScanner(rulesDir string) (*yara.Scanner, error) {
	compiler, err := y.loadRules(rulesDir)
	if err != nil {
		return nil, err
	}

	rules, err := compiler.GetRules()
	if err != nil {
		return nil, err
	}

	y.logger.Infow("Yara rules loaded", "Rules", len(rules.GetRules()))

	scanner, err := yara.NewScanner(rules)
	if err != nil {
		return nil, err
	}

	return scanner, nil
}

func (y *YaraScanner) loadRules(rulesDir string) (compiler *yara.Compiler, err error) {
	compiler, err = yara.NewCompiler()
	if err != nil {
		return nil, errors.New("failed to initialize yara compiler")
	}

	err = filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			y.logger.Errorw("failed to parse directory", "error", err, "path", path)
			return nil
		}

		if info.Mode().IsRegular() {
			if err := y.loadSingleRule(path, compiler); err != nil {
				y.logger.Errorw("failed to load single rule", "error", err, "path", path)
			}
		}

		return nil
	})

	if err != nil {
		y.logger.Errorw("failed to load rules", "error", err)
	}

	return compiler, nil
}

func (y *YaraScanner) loadSingleRule(path string, compiler *yara.Compiler) error {
	y.logger.Infow("Loading rule.", "rule", path)

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to load rule. err: %w", err)
	}

	// TODO use internal directory as namespace
	// Eg.: rules/ransomware/ruleA.yar (namespace ransomware)
	err = compiler.AddFile(f, "rules")
	if err != nil {
		return fmt.Errorf("failed to compile rule. err: %w", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to close file. err: %w", err)
	}

	return nil
}

func (y *YaraScanner) Scan(ctx context.Context, sc scanContext) (entities.ScanResult, error) {
	if y.scanner == nil {
		return entities.ScanResult{}, nil
	}

	// Loads chunk to memory, because using ScanBinary requires lots of main memory.
	stop := false
	matchCounter := 0
	result := entities.ScanResult{Entropy: entities.GenerateEntropyBuckets([9]int{})}

	file, err := sc.Storage.Open(sc.Filename)
	if err != nil {
		y.logger.Errorw("couldn't open file to calculate entropy", "error", err, "filename", sc.Filename)
		return result, err
	}
	defer file.Close()

	for !stop {
		var matches yara.MatchRules
		n, err := file.Read(sc.Buffer)

		if err != nil && !errors.Is(err, io.EOF) {
			y.logger.Errorw("failed reading reader", "error", err)
			result.Matches += matchCounter
		}

		if errors.Is(err, io.EOF) {
			stop = true
		}

		err = y.scanner.SetCallback(&matches).ScanMem(sc.Buffer[:n])
		if err != nil {
			result.Matches += matchCounter
			return result, err
		}

		matchCounter += len(matches)
	}

	result.Matches += matchCounter

	return result, nil
}
