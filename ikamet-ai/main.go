package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// --- 1. GELİŞMİŞ HESAPLAMA FONKSİYONU ---
func hesaplaIkametUcreti(args string) (string, error) {
	var params struct {
		Uyruk string `json:"uyruk"`
		Yas   int    `json:"yas"`
		Sure  int    `json:"sure_ay"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}

	// Uyruk verisini standardize edelim
	uyruk := strings.ToLower(strings.TrimSpace(params.Uyruk))

	// 1. Harç Bedeli (Ülkelere göre değiştiğini simüle ediyoruz)
	aylikHarc := 80.0
	kartBedeli := 565.0 // 2024 İkamet Kartı Bedeli örneği

	// Bazı ülkelerin harç muafiyeti veya farklı tarifesi olabilir
	if uyruk == "suriye" || uyruk == "türkmenistan" {
		aylikHarc = 40.0
	} else if uyruk == "almanya" {
		aylikHarc = 90.0 // Temsili
	}
	harcToplami := (float64(params.Sure) * aylikHarc) + kartBedeli

	// 2. Özel Sağlık Sigortası (Yaşa göre algoritma)
	sigortaBedeli := 0.0
	if params.Yas >= 18 && params.Yas <= 65 {
		if params.Yas < 25 {
			sigortaBedeli = 1500.0
		} else if params.Yas < 45 {
			sigortaBedeli = 2500.0
		} else {
			sigortaBedeli = 4500.0
		}
	} else {
		// 18 yaş altı veya 65 yaş üstü için özel sigorta zorunluluğu yoktur (istisnalar hariç)
		sigortaBedeli = 0.0
	}

	// 3. Acente Hizmet Bedeli
	hizmetBedeli := 2000.0

	toplam := harcToplami + sigortaBedeli + hizmetBedeli

	sonuc := fmt.Sprintf(`{
		"durum": "basarili",
		"detaylar": {
			"uyruk": "%s",
			"yas": %d,
			"sure_ay": %d
		},
		"maliyetler": {
			"devlet_harci_ve_kart": %.2f,
			"saglik_sigortasi": %.2f,
			"acente_hizmet_bedeli": %.2f,
			"toplam_tutar": %.2f,
			"para_birimi": "TL"
		},
		"not": "18 yaş altı ve 65 yaş üstü için sigorta bedeli 0 TL olarak hesaplanmıştır."
	}`, strings.ToTitle(uyruk), params.Yas, params.Sure, harcToplami, sigortaBedeli, hizmetBedeli, toplam)

	return sonuc, nil
}

// --- 2. EVRAK BİLGİ BANKASI ---
func getirEvrakListesi(args string) (string, error) {
	bilgiBankasi := `
	Tüm başvurular için ortak evraklar: Pasaport, 4 adet biyometrik fotoğraf, Adres kayıt belgesi.
	Turistik: Gelir beyanı ve turizm planı.
	Öğrenci: Üniversiteden alınmış aktif öğrenci belgesi.
	Aile: Destekleyicinin gelir belgesi ve adli sicil kaydı.
	Eksik Evrak Kuralı: Göç idaresi eksik evrak bildirirse 30 gün ek süre tanınır.`
	return bilgiBankasi, nil
}

func main() {
	ctx := context.Background()

	llm, err := ollama.New(ollama.WithModel("llama3.1"))
	if err != nil {
		log.Fatalf("LLM başlatılamadı: %v", err)
	}

	tools := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "hesapla_ikamet_ucreti",
				Description: "İkamet maliyetini hesaplar. DİKKAT: Bu fonksiyonu çağırmadan önce kullanıcının UYRUK, YAŞ ve SÜRE bilgilerinin üçünü de kesinlikle bilmelisin.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"uyruk":   map[string]any{"type": "string", "description": "Örn: Almanya, Rusya, İran"},
						"yas":     map[string]any{"type": "integer", "description": "Kişinin yaşı"},
						"sure_ay": map[string]any{"type": "integer", "description": "Kaç aylık ikamet istendiği"},
					},
					"required": []string{"uyruk", "yas", "sure_ay"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "getir_evrak_listesi",
				Description: "İkamet başvuru evraklarını getirir.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ikamet_turu": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	fmt.Println("🤖 Akıllı Acente AI Başladı. (Çıkmak için 'exit')")
	fmt.Println(strings.Repeat("-", 60))

	scanner := bufio.NewScanner(os.Stdin)
	var mesajGecmisi []llms.MessageContent

	// PROMPT MÜHENDİSLİĞİ: Modelin aklını yöneteceğimiz yer burası.
	sistemTalimati := `Sen profesyonel ve kibar bir Türkiye ikamet/vize danışmanlık asistanısın.

GÖREV 1: FİYAT HESAPLAMA
Kullanıcı ikamet fiyatı, maliyeti veya ücreti sorduğunda hesaplama yapman gerekir. 
ANCAK hesaplama yapabilmek için ŞU 3 BİLGİYE KESİNLİKLE İHTİYACIN VAR:
1. Uyruk (Hangi ülkenin vatandaşı?)
2. Yaş (Kişi kaç yaşında?)
3. Süre (Kaç aylık veya yıllık ikamet istiyor?)

KURAL: Eğer kullanıcı fiyat sorarsa, hemen bu 3 bilginin tam olup olmadığını kontrol et. 
Eksik bilgi varsa ASLA 'hesapla_ikamet_ucreti' aracını KULLANMA. Önce kibarca eksik olan bilgileri kullanıcıya sor (Örn: "Size net bir fiyat çıkarabilmem için lütfen uyruğunuzu, yaşınızı ve kaç yıllık ikamet istediğinizi belirtebilir misiniz?"). 
Sadece bu 3 bilgiyi de elde ettiğinde aracı çalıştır.

GÖREV 2: EVRAK BİLGİSİ
Evrak sorulursa 'getir_evrak_listesi' aracını kullan ve sadece oradan gelen bilgiyi ver.`

	mesajGecmisi = append(mesajGecmisi, llms.TextParts(llms.ChatMessageTypeSystem, sistemTalimati))

	for {
		fmt.Print("\nMüşteri: ")
		if !scanner.Scan() {
			break
		}
		userInput := scanner.Text()
		if strings.ToLower(userInput) == "exit" {
			break
		}

		mesajGecmisi = append(mesajGecmisi, llms.TextParts(llms.ChatMessageTypeHuman, userInput))

		resp, err := llm.GenerateContent(ctx, mesajGecmisi, llms.WithTools(tools))
		if err != nil {
			log.Printf("Hata: %v", err)
			continue
		}

		cevap := resp.Choices[0]

		if len(cevap.ToolCalls) > 0 {
			for _, toolCall := range cevap.ToolCalls {
				fmt.Printf("⚙️  [Araç Tetiklendi: %s]\n", toolCall.FunctionCall.Name)

				var toolResult string
				if toolCall.FunctionCall.Name == "hesapla_ikamet_ucreti" {
					toolResult, _ = hesaplaIkametUcreti(toolCall.FunctionCall.Arguments)
				} else if toolCall.FunctionCall.Name == "getir_evrak_listesi" {
					toolResult, _ = getirEvrakListesi(toolCall.FunctionCall.Arguments)
				}

				mesajGecmisi = append(mesajGecmisi, llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: toolCall.ID,
							Name:       toolCall.FunctionCall.Name,
							Content:    toolResult,
						},
					},
				})
			}

			resp, err = llm.GenerateContent(ctx, mesajGecmisi, llms.WithTools(tools))
			if err != nil {
				log.Printf("Hata: %v", err)
				continue
			}
			cevap = resp.Choices[0]
		}

		fmt.Printf("🤖 AI: %s\n", cevap.Content)
		mesajGecmisi = append(mesajGecmisi, llms.TextParts(llms.ChatMessageTypeAI, cevap.Content))
	}
}
