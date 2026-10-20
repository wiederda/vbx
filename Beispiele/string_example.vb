#use string

' ============================
' String Beispiele für VBX
' ============================

Print EncodeBase64("Hello")          ' → "SGVsbG8="
Print DecodeBase64("SGVsbG8=")       ' → "Hello"

Print string.HexEncode("abc")               ' → "616263"
Print string.HexDecode("616263")            ' → "abc"

Print URLEncode("a b+c")            ' → "a+b%2Bc"
Print URLDecode("a+b%2Bc")          ' → "a b+c"

Print ToInt("42")                    ' → 42
Print ToFloat("3.14")                ' → 3.14
Print ToBool("true")                 ' → 1
Print ToString(123.4)                ' → "123.4"

Print Length("Hello")                    ' → 5
Print Trim("  Hello ")                ' → "Hello"
Print TrimStart("abcHello", "ab")      ' → "cHello"
Print TrimEnd("Helloabc", "bc")     ' → "Hello"
Print ToLower("HeLLo")                ' → "hello"
Print ToUpper("HeLLo")                ' → "HELLO"
Print Replace("a-b-c", "-", "_")     ' → "a_b_c"
Print string.Repeat("Ha", 3)                 ' → "HaHaHa"
Print Contains("Hello", "ll")         ' → 1
Print IndexOf("Hello", "ll")          ' → 2
Print Left("Hello", 2)                ' → "He"
Print Right("Hello", 2)               ' → "lo"
Print Substring("Hello", 1, 3)          ' → "ell"

Print MD5("abc")                      ' → "900150983cd24fb0d6963f7d28e17f72"
Print SHA1("abc")                     ' → "a9993e364706816aba3e25717850c26c9cd0d89d"
Print SHA256("abc")                   ' → "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

Print InStr("Hello", "ll")            ' → 3
Print Mid("Hello", 2, 2)              ' → "el"
Print string.Reverse("abc")                  ' → "cba"
Print string.PadLeft("Hi", 5, "*")           ' → "***Hi"
Print string.PadRight("Hi", 5, "*")          ' → "Hi***"

' StrComp: default binary (case-sensitive)
Print string.StrComp("Apfel", "apfel")  ' -1
Print string.StrComp("Apfel", "Banane") ' -1

' StrComp: text compare (case-insensitive)
Print string.StrComp("Apfel", "apfel", 1) ' 0

' CompareText: immer case-insensitive
Print string.CompareText("Apfel", "apfel") ' 0
Print string.CompareText("Banane", "Apfel") ' 1
