Print "=== Array Create ==="
Dim arr = array.Create("a", "b", "c")
Dim arry = array.Create(3)
Print array.Join(arr, ",")

Print "=== Count / Bounds ==="
Print "Count: " + array.Count(arr)
Print "LBound: " + array.LBound(arr)
Print "UBound: " + array.UBound(arr)

Print "=== First / Last ==="
Print "First: " + array.First(arr)
Print "Last: " + array.Last(arr)

Print "=== Append ==="
Dim arr2 = array.Append(arr, "d")
Print array.Join(arr2, ",")

Print "=== Insert ==="
Dim arr3 = array.Insert(arr2, 1, "x")
Print array.Join(arr3, ",")

Print "=== Remove ==="
Dim arr4 = array.Remove(arr3, 2)
Print array.Join(arr4, ",")

Print "=== SetIndex ==="
Dim arr5 = array.SetIndex(arr4, 0, "z")
Print array.Join(arr5, ",")

Print "=== Contains ==="
Print array.Contains(arr5, "x")
Print array.Contains(arr5, "notfound")

Print "=== IndexOf ==="
Print array.IndexOf(arr5, "x")

Print "=== Clone ==="
Dim clone = array.Clone(arr5)
Print array.Join(clone, ",")

Print "=== Reverse ==="
Dim rev = array.Reverse(arr5)
Print array.Join(rev, ",")

Print "=== Sort (Strings) ==="
Dim unsorted = array.Create("c", "a", "b")
Dim sorted = array.Sort(unsorted)
Print array.Join(sorted, ",")

Print "=== Sort (Numbers) ==="
Dim nums = array.Create(5, 1, 3)
Dim numsSorted = array.Sort(nums)
Print array.Join(numsSorted, ",")

Print "=== Unique ==="
Dim dup = array.Create("a", "b", "a", "c", "b")
Dim unique = array.Unique(dup)
Print array.Join(unique, ",")

Print "=== Merge ==="
Dim merged = array.Merge(arr5, unique)
Print array.Join(merged, ",")

Print "=== IsEmpty ==="
Dim emptyArr = array.Create()
Print array.IsEmpty(emptyArr)
Print array.IsEmpty(arr5)

Print "=== Clear ==="
Dim cleared = array.Clear(arr5)
Print array.Count(cleared)

Print "=== Split ==="
Dim parts = array.Split("one,two,three", ",")
Print array.Join(parts, "|")
