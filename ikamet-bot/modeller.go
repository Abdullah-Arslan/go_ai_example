package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY bulunamadı. Terminalde tanımladığından emin ol.")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal("İstemci oluşturulamadı:", err)
	}
	defer client.Close()

	fmt.Println("Senin API Anahtarının Erişebildiği Modeller:")
	fmt.Println(strings.Repeat("-", 40))

	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal("Modeller çekilirken hata oluştu:", err)
		}

		// Sadece içerik üretebilen (generateContent) modelleri filtreleyelim
		if strings.Contains(m.Name, "gemini") {
			fmt.Printf("- %s\n", strings.ReplaceAll(m.Name, "models/", ""))
		}
	}
}
