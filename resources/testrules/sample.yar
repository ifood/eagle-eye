/*
    Sample YARA rule to use in the test.
*/

rule sample_rule : ransomware
{
    meta:
        description = "Sample rule to be used in test"

    strings:
        $a = "IAMAMALWARE"

    condition:
        $a
}