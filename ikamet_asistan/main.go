package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// hesaplaIkametUcreti, yapay zekadan gelen verileri alıp şemadaki kurallara göre fiyat ve belge listesi üretir
func hesaplaIkametUcreti(args map[string]any) map[string]any {
	ikametTuru, _ := args["ikamet_turu"].(string)

	harcBedeliYillik := 1500
	kartUcreti := 565
	hizmetBedeli := 2000
	sigortaBedeli := 3000

	if ikametTuru == "Uzun Dönem" {
		calismaIzniVarMi, ok := args["calisma_izni_var_mi"].(bool)
		if ok && !calismaIzniVarMi {
			return map[string]any{
				"hata": "Çalışma izniniz olmadığı için doğrudan uzmanlarımızla görüşmeniz gerekmektedir. Lütfen iletişime geçin: +90 555 123 45 67",
			}
		}

		dahiller := []string{"Randevu Alma", "Harç Bedeli", "Kart Ücreti", "Hizmet Bedeli"}
		evraklar := []string{"Pasaport Kopyası", "Biyometrik Fotoğraf", "Çalışma İzni Belgesi", "Adres Kayıt Belgesi"}

		return map[string]any{
			"toplam_ucret":     "5500 ₺",
			"fiyata_dahiller":  strings.Join(dahiller, ", "),
			"gerekli_evraklar": strings.Join(evraklar, ", "),
			"paylasim_mesaji":  "Fiyatlandırma ve başvuru süreci detayları için WhatsApp hattımızdan bize ulaşın.",
		}
	}

	sureYil := 1
	if s, ok := args["sure_yil"].(float64); ok {
		sureYil = int(s)
	}

	toplamUcret := kartUcreti + hizmetBedeli + (harcBedeliYillik * sureYil)
	sigortaEklensinMi := true

	if ikametTuru == "Aile İkameti" {
		esDurumu, _ := args["es_durumu"].(float64)
		durumKodu := int(esDurumu)
		if durumKodu == 1 || durumKodu == 3 {
			sigortaEklensinMi = false
		}
	}

	if ikametTuru == "Öğrenci İkameti" {
		sigortaEklensinMi = true
	}

	dahilOlanlar := []string{"Randevu Alma", "Harç Bedeli", "Kart Ücreti", "Hizmet Bedeli"}

	if sigortaEklensinMi {
		toplamUcret += sigortaBedeli * sureYil
		dahilOlanlar = append(dahilOlanlar, "Sigorta")
	}

	var evraklar []string
	switch ikametTuru {
	case "Kısa Dönem":
		evraklar = []string{"Pasaport Fotokopisi", "4 Adet Biyometrik Fotoğraf", "Kira Kontratı (Noter Onaylı)", "Vergi Numarası", "Sigorta Poliçesi"}
	case "Aile İkameti":
		evraklar = []string{"Pasaport Fotokopisi", "4 Adet Biyometrik Fotoğraf", "Evlilik Cüzdanı (Apostilli)", "Destekleyicinin Kimliği", "Vukuatlı Nüfus Kayıt Örneği"}
	case "Öğrenci İkameti":
		evraklar = []string{"Pasaport Fotokopisi", "4 Adet Biyometrik Fotoğraf", "Öğrenci Belgesi (Aktif)", "Yurt/Kira Sözleşmesi"}
	}

	// HATA BURADA ÇÖZÜLDÜ: Listeleri strings.Join ile tek bir metne çeviriyoruz
	return map[string]any{
		"toplam_ucret":     fmt.Sprintf("%d ₺", toplamUcret),
		"fiyata_dahiller":  strings.Join(dahilOlanlar, ", "),
		"gerekli_evraklar": strings.Join(evraklar, ", "),
		"paylasim_mesaji":  "Fiyatlandırma ve başvuru süreci detayları için WhatsApp hattımızdan bize ulaşın.",
	}
}

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Lütfen GEMINI_API_KEY ortam değişkenini ayarlayın.")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("İstemci oluşturulamadı: %v", err)
	}
	defer client.Close()

	// MODEL İSMİ GÜNCELLENDİ
	model := client.GenerativeModel("gemini-2.5-flash")

	model.SystemInstruction = genai.NewUserContent(genai.Text(`
		Sen profesyonel bir İkamet İzni Asistanısın. Kullanıcıyla adım adım ilgileneceksin. 
		Asla tüm soruları aynı anda sorma. Kullanıcının cevabına göre sıradaki soruyu sor.
		
		İzleyeceğin adımlar sırasıyla şunlardır:
		1. ADIM: "İlk başvuru mu yapmak istiyorsunuz, yoksa uzatma başvurusu mu?" diye sor. Cevabı bekle.
		2. ADIM: "İkamet çeşidinizi seçiniz: Kısa Dönem İkameti, Aile İkameti, Öğrenci İkameti, Uzun Dönem İkameti" diye sor. Cevabı bekle.
		3. ADIM: İkamet türü seçildikten sonra ilgili dalı takip et:
			- Eğer "Kısa Dönem" ise: Uyruğunu, Doğum Yılını ve İkamet Süresini (1 Yıl veya 2 Yıl) sor.
			- Eğer "Aile İkameti" ise: Uyruğunu, Doğum Yılını, İkamet Süresini (1, 2 veya 3 Yıl) VE Eşinin durumunu sor (Şu 4 seçenekten birini seçmesini iste: 1-Eşi Türk/SGK var, 2-Eşi Türk/SGK yok, 3-Eşi Yabancı/SGK var, 4-Eşi Yabancı/SGK yok).
			- Eğer "Öğrenci İkameti" ise: Doğum Yılını ve İkamet Süresini (1 veya 2 Yıl) sor.
			- Eğer "Uzun Dönem İkameti" ise: "Çalışma izniniz var mı?" diye sor.
		
		4. ADIM: Kullanıcı kendi senaryosuna ait tüm verileri sağladığında, KESİNLİKLE 'ucret_ve_evrak_hesapla' fonksiyonunu çağır.
		5. ADIM: Fonksiyondan gelen Toplam Ücreti, Fiyata Dahil Olan Hizmetleri ve Gerekli Evraklar listesini kullanıcıya şık bir şekilde sun ve sonunda WhatsApp paylaşım yönlendirmesi yap.
	`))

	hesaplaTool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "ucret_ve_evrak_hesapla",
			Description: "İstenen ikamet türüne göre toplam fiyatı, dahil olan hizmetleri ve evrakları hesaplar.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"ikamet_turu":         {Type: genai.TypeString, Description: "Kısa Dönem, Aile İkameti, Öğrenci İkameti, Uzun Dönem seçeneklerinden biri."},
					"uyruk":               {Type: genai.TypeString, Description: "Vatandaşı olduğu ülke (Varsa)."},
					"dogum_yili":          {Type: genai.TypeInteger, Description: "Doğum yılı (Örn: 1990)."},
					"sure_yil":            {Type: genai.TypeInteger, Description: "Talep edilen ikamet süresi (1, 2 veya 3)."},
					"es_durumu":           {Type: genai.TypeInteger, Description: "Sadece Aile ikameti için. 1, 2, 3 veya 4 değeri."},
					"calisma_izni_var_mi": {Type: genai.TypeBoolean, Description: "Sadece Uzun Dönem ikameti için. Evet ise true, hayır ise false."},
				},
				Required: []string{"ikamet_turu"},
			},
		}},
	}
	model.Tools = []*genai.Tool{hesaplaTool}

	session := model.StartChat()
	fmt.Println("🚀 İkamet Otomasyonu Başlatıldı! (Çıkmak için 'q' yazın)")
	fmt.Println("---------------------------------------------------------")

	resp, err := session.SendMessage(ctx, genai.Text("Merhaba, sisteme giriş yaptım. Bana ilk adımı sor."))
	if err != nil {
		log.Fatalf("Sisteme bağlanırken bir hata oluştu: %v", err)
	}
	handleResponse(ctx, session, resp)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nSen: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "q" {
			break
		}

		resp, err := session.SendMessage(ctx, genai.Text(input))
		if err != nil {
			log.Printf("Hata: %v", err)
			continue
		}
		handleResponse(ctx, session, resp)
	}
}

func handleResponse(ctx context.Context, session *genai.ChatSession, resp *genai.GenerateContentResponse) {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		fmt.Println("Asistan: (Sistemden geçerli bir yanıt alınamadı. Lütfen tekrar deneyin.)")
		return
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			fmt.Println("Asistan:\n" + string(v))

		case genai.FunctionCall:
			if v.Name == "ucret_ve_evrak_hesapla" {
				fmt.Println("\n[Sistem: Bilgiler alındı, fiyat ve evraklar hesaplanıyor...]")

				sonuc := hesaplaIkametUcreti(v.Args)

				resp2, err := session.SendMessage(ctx, genai.FunctionResponse{
					Name:     v.Name,
					Response: sonuc,
				})

				if err != nil {
					log.Println("Fonksiyon sonucu modele iletilemedi:", err)
					return
				}

				handleResponse(ctx, session, resp2)
			}
		}
	}
}
