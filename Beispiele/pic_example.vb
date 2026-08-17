#use picture

picture.Convert("pic/bild.webp", "pic/input.png", "png")

' Einzelbild konvertieren in JPG, 80% Qualität
picture.Convert("pic/input.png", "pic/output.jpg", "jpg", "20", "20", "20")

' Einzelbild konvertieren in PNG, Größe 100*100
picture.Convert("pic/input.png", "pic/output.png", "png", "50", "50")

' Einzelbild in PNG skalieren auf 200x200
picture.Resize("pic/input.png", "pic/resized.png", "200", "200")

' Einzelbild zuschneiden auf 100x100
picture.Crop("pic/input.jpg", "pic/cropped.jpg", "100", "100")

' Einzelbild in ICO erstellen (16, 32, 48, 64, 128, 256 Pixel)
picture.Convert("pic/Icon.png", "pic/app.ico", "ico", "16,32,48,64,128,256")

' Alles außer ICO in WebP
picture.ConvertAll("pic", "pic/convert", "webp")

' Nur WebP-Dateien nach JPG konvertieren, Größe 100x100, Qualität 80
picture.ConvertAll("pic", "pic/convert", "jpg", "webp", "100", "100", "80")

' Nur PNG- und JPEG-Dateien nach WebP
picture.ConvertAll("pic", "pic/convert", "webp", "png,jpg")

' Alle außer ICO nach ICO konvertieren (16, 32, 64 Pixel)
picture.ConvertAll("pic", "pic/convert", "ico", "16,32,64")


