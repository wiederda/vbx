#use git

' ============================
' Git Beispiele für VBX
' ============================

Dim status, logEntries, hash, currentBranch

Print "=== git.IsRepo ==="
Print "Ist /home/vbx/test-repo ein Repo: " & git.IsRepo("/home/vbx/test-repo")
Print ""

Print "=== git.Clone ==="
' Ohne path -> Zielverzeichnis wird aus der URL abgeleitet ("vbx-demo")
git.Clone("https://github.com/user/vbx-demo.git", "/home/vbx/vbx-demo")
Print "Repository geklont nach /home/vbx/vbx-demo"
Print ""

Print "=== git.CloneWithToken ==="
' git.CloneWithToken("https://github.com/user/private-repo.git", ghp_xxx, "/home/vbx/private-repo")
Print "(auskommentiert - benötigt echtes Token)"
Print ""

Print "=== git.CloneWithKey ==="
' git.CloneWithKey("git@github.com:user/private-repo.git", "/home/vbx/.ssh/id_ed25519", "/home/vbx/private-repo", "/home/vbx/.ssh/known_hosts")
Print "(auskommentiert - benötigt echten SSH-Key)"
Print ""

Print "=== git.CurrentBranch ==="
currentBranch = git.CurrentBranch("/home/vbx/vbx-demo")
Print "Aktueller Branch: " & currentBranch
Print ""

Print "=== git.Status ==="
status = git.Status("/home/vbx/vbx-demo")
Print "Geändert:"
For i = 0 To array.UBound(status("modified"))
    Print "  " & status("modified")(i)
Next
Print "Neu:"
For i = 0 To array.UBound(status("untracked"))
    Print "  " & status("untracked")(i)
Next
Print ""

Print "=== git.Add ==="
file.WriteText("/home/vbx/vbx-demo/notiz.txt", "Erste Notiz")
git.Add("notiz.txt", "/home/vbx/vbx-demo")
Print "notiz.txt gestaged"
Print ""

Print "=== git.Commit ==="
hash = git.Commit("Notiz hinzugefügt", "/home/vbx/vbx-demo")
Print "Commit erstellt: " & hash
Print ""

Print "=== git.Log ==="
logEntries = git.Log(5, "/home/vbx/vbx-demo")
For i = 0 To array.UBound(logEntries)
    Print logEntries(i)("hash") & " | " & logEntries(i)("author") & " | " & logEntries(i)("message")
Next
Print ""

Print "=== git.Diff ==="
' Diff des letzten Commits gegen HEAD (hier identisch, nur als Beispiel)
Print git.Diff(hash, "", "/home/vbx/vbx-demo")
Print ""

Print "=== git.Checkout (neuer Branch) ==="
git.Checkout("feature-notiz", true, "/home/vbx/vbx-demo")
Print "Auf Branch 'feature-notiz' gewechselt"
Print ""

Print "=== git.Checkout (zurück zu main) ==="
git.Checkout("main", false, "/home/vbx/vbx-demo")
Print "Zurück auf 'main'"
Print ""

Print "=== git.Reset ==="
git.Add("notiz.txt", "/home/vbx/vbx-demo")
git.Reset("mixed", "/home/vbx/vbx-demo")
Print "Staging zurückgesetzt (mixed)"
Print ""

Print "=== git.ResetHard ==="
' Vorsicht: alle lokalen Änderungen nach diesem Commit gehen verloren
git.ResetHard(hash, "/home/vbx/vbx-demo")
Print "Hart zurückgesetzt auf " & hash
Print ""

Print "=== git.Pull ==="
git.Pull("/home/vbx/vbx-demo")
Print "Pull durchgeführt"
Print ""

Print "=== git.Fetch ==="
git.Fetch("/home/vbx/vbx-demo")
Print "Fetch durchgeführt"
Print ""

Print "=== git.Push ==="
' Ohne Auth-Parameter -> nur für unauthentifizierte Remotes
git.Push("/home/vbx/vbx-demo")
Print "Push durchgeführt"
Print ""

Print "=== git.Push mit SSH-Key ==="
' git.Push("/home/vbx/vbx-demo", "/home/vbx/.ssh/id_ed25519", "/home/vbx/.ssh/known_hosts")
Print "(auskommentiert - benötigt echten SSH-Key)"
Print ""

Print "=== git.QuickPush ==="
file.WriteText("/home/vbx/vbx-demo/notiz2.txt", "Zweite Notiz")
hash = git.QuickPush("Zweite Notiz hinzugefügt", ".", "/home/vbx/vbx-demo")
Print "Add + Commit + Push in einem: " & hash
Print ""

Print "=== git.Remove ==="
git.Remove("notiz2.txt", "/home/vbx/vbx-demo")
Print "notiz2.txt aus Index und Arbeitsverzeichnis entfernt"
Print ""

Print "=== Test abgeschlossen ==="
