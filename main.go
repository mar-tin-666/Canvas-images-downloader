package main

import (
	"bufio"
	"fmt"
	"image"
	_ "image/jpeg" // Obsługa formatu JPEG
	_ "image/png"  // Obsługa formatu PNG
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

func main() {
	var baseURL string

	// Pobranie linku od użytkownika
	fmt.Print("Podaj link do pliku: ")
	fmt.Scanln(&baseURL)

	// Pytanie o złożenie obrazków do PDF
	fmt.Print("Czy chcesz złożyć obrazki do PDF? (Y/N, domyślnie N): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToUpper(response))

	// Dopasowanie linku do wyrażenia regularnego
	re := regexp.MustCompile(`(.*/)([^/]*?)(\d{6})(\.[a-z]+)$`)
	matches := re.FindStringSubmatch(baseURL)
	if matches == nil {
		fmt.Println("Link nie pasuje do wymaganego formatu.")
		return
	}
	basePath, prefix, extension := matches[1], matches[2], matches[4]

	// Ustawienie folderu zapisu w /downloaded/
	downloadDir := filepath.Join("downloaded", strings.TrimSuffix(prefix, "_"))
	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		fmt.Println("Nie można utworzyć folderu:", err)
		return
	}

	var imageFiles []string
	// Pobieranie obrazków, zaczynając od 1
	for i := 1; ; i++ {
		fileNum := fmt.Sprintf("%06d", i)
		fileURL := fmt.Sprintf("%s%s%s%s", basePath, prefix, fileNum, extension)
		filePath := filepath.Join(downloadDir, fmt.Sprintf("%s%s%s", prefix, fileNum, extension))

		resp, err := http.Get(fileURL)
		if err != nil {
			fmt.Println("Błąd przy pobieraniu obrazu:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			fmt.Println("Brak więcej obrazów do pobrania. Zakończono.")
			break
		} else if resp.StatusCode != http.StatusOK {
			fmt.Printf("Błąd HTTP %d przy pobieraniu %s\n", resp.StatusCode, fileURL)
			break
		}

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println("Nie można utworzyć pliku:", err)
			return
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			fmt.Println("Błąd przy zapisywaniu obrazu:", err)
			return
		}

		fmt.Printf("Zapisano: %s\n", filePath)
		imageFiles = append(imageFiles, filePath)
	}

	// Generowanie PDF, jeśli użytkownik wybrał tak / yes
	fmt.Print("\n---\n\n")
	if response == "Y" || response == "T" {
		pdfPath := filepath.Join(downloadDir, fmt.Sprintf("%s.pdf", strings.TrimSuffix(prefix, "_")))
		err := createPDF(imageFiles, pdfPath)
		if err != nil {
			fmt.Println("Błąd przy tworzeniu PDF:", err)
		} else {
			fmt.Printf("Plik PDF zapisano jako: %s\n", pdfPath)
		}
	} else {
		fmt.Println("Obrazki nie zostały złożone do PDF.")
	}
}

// Funkcja tworząca plik PDF z listy obrazków
func createPDF(imageFiles []string, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	for _, file := range imageFiles {
		// Otwarcie pliku graficznego i sprawdzenie wymiarów
		imgFile, err := os.Open(file)
		if err != nil {
			fmt.Println("Błąd przy otwieraniu obrazu:", err)
			return err
		}
		img, _, err := image.DecodeConfig(imgFile)
		imgFile.Close() // Zamknięcie pliku
		if err != nil {
			fmt.Println("Błąd przy odczycie rozmiaru obrazu:", err)
			return err
		}

		// Przekształcenie wymiarów obrazka na milimetry
		imgWidthMM := float64(img.Width) * 0.264583
		imgHeightMM := float64(img.Height) * 0.264583

		// Tworzenie strony o rozmiarze zgodnym z rozmiarem obrazka
		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: imgWidthMM, Ht: imgHeightMM})
		options := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		}

		// Wstawienie obrazka w oryginalnym rozmiarze
		pdf.ImageOptions(file, 0, 0, imgWidthMM, imgHeightMM, false, options, 0, "")
	}
	return pdf.OutputFileAndClose(outputPath)
}
