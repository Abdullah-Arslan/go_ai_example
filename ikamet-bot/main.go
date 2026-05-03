package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Hesaplama için gerekli parametreleri tutan Struct
type Hesaplamaİstegi struct {
	Yas          int    `json:"yas"`
	Uyruk        string `json:"uyruk"`
	IkametSuresi int    `json:"ikamet_suresi_yil"` // Yıl bazında
}

// Hesaplama sonucunu döndüreceğimiz Struct
type HesaplamaSonucu struct {
	HarBedeli   float64 `json:"harc_bedeli_usd"`
	Sigorta     float64 `json:"sigorta_tl"`
	Hizmet      float64 `json:"hizmet_bedeli_tl"`
	ToplamMesaj string  `json:"toplam_mesaj"`
}

// 1. ADIM: KESİN HESAPLAMA FONKSİYONU (Function Calling için arka plan mantığı)
func ikametUcretiHesapla(istek Hesaplamaİstegi) HesaplamaSonucu {
	// Bu kısımda normalde veritabanına veya güncel kur API'sine bağlanılır.
	// Örnek sabit fiyatlandırma mantığı:

	sigortaFiyati := 1500.0
	if istek.Yas > 65 {
		sigortaFiyati = 4500.0 // 65 yaş üstü sigorta daha pahalı
	}

	harcBedeliUSD := 80.0 * float64(istek.IkametSuresi)
	if strings.ToLower(istek.Uyruk) == "sirbistan" {
		harcBedeliUSD = 0 // Sırbistan vatandaşları harçtan muaf (örnek kural)
	}

	hizmetBedeli := 2500.0 // Acentenin sabit hizmet bedeli

	return HesaplamaSonucu{
		HarBedeli: harcBedeliUSD,
		Sigorta:   sigortaFiyati,
		Hizmet:    hizmetBedeli,
		ToplamMesaj: fmt.Sprintf("%d yıllık ikamet için toplam %v USD harç, %v TL sigorta ve %v TL hizmet bedeli çıkmaktadır.",
			istek.IkametSuresi, harcBedeliUSD, sigortaFiyati, hizmetBedeli),
	}
}

// 2. ADIM: RAG SİMÜLASYONU (Vektör veritabanından bilgi getirme)
func ragBaglaminiGetir(kullaniciSorusu string) string {
	// Gerçek bir senaryoda burada Pinecone veya pgvector sorgusu yapılır.
	// Şimdilik soruya göre statik metin dönüyoruz.
	if strings.Contains(strings.ToLower(kullaniciSorusu), "evrak") {
		return "RAG BİLGİSİ: Turistik ikamet için gerekli evraklar: 1) Pasaport fotokopisi, 2) 4 adet biyometrik fotoğraf, 3) Geçerli sağlık sigortası, 4) Adres kayıt belgesi veya noter onaylı kira sözleşmesi. Göç idaresi eksik evrak durumunda 30 gün ek süre tanır."
	}
	return "RAG BİLGİSİ: İkamet başvuru süreçleri ortalama 90 gün sürmektedir. Başvuru yapıldıktan sonra ülkede yasal olarak kalınabilir."
}

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY ortam değişkeni bulunamadı.")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal("İstemci oluşturulamadı:", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	// 3. ADIM: YAPAY ZEKAYA FONKSİYONU TANITMA (Tool Declaration)
	hesaplaAraci := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "ikamet_ucreti_hesapla",
			Description: "Yabancının ikamet başvuru ücretlerini, sigorta ve harç bedelini hesaplar. Yaş, uyruk ve istenen yıl zorunludur.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"yas": {
						Type:        genai.TypeInteger,
						Description: "Yabancının yaşı (örneğin 35).",
					},
					"uyruk": {
						Type:        genai.TypeString,
						Description: "Yabancının vatandaşı olduğu ülke.",
					},
					"ikamet_suresi_yil": {
						Type:        genai.TypeInteger,
						Description: "Kaç yıllık ikamet isteniyor (1 veya 2).",
					},
				},
				Required: []string{"yas", "uyruk", "ikamet_suresi_yil"},
			},
		}},
	}
	model.Tools = []*genai.Tool{hesaplaAraci}

	// Kullanıcıdan gelen örnek soru
	kullaniciMesaji := "Merhaba, Sırbistan vatandaşıyım. 45 yaşındayım ve 2 yıllık ikamet almak istiyorum. Ne kadar öderim ve hangi evrakları hazırlamam lazım?"

	// RAG Bağlamını oluşturup sisteme veriyoruz
	ragBilgisi := ragBaglaminiGetir(kullaniciMesaji)
	sistemTalimati := fmt.Sprintf(`Sen profesyonel bir acente asistanısın. 
	Kurallar:
	1. Kesinlikle kendin matematik hesabı yapma, her zaman ikamet_ucreti_hesapla aracını kullan.
	2. Sana sağlanan şu resmi bilgilere dayanarak evrak sorularına cevap ver: %s
	3. Müşteriye nazik, güven verici ve kurumsal bir dille hitap et.`, ragBilgisi)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(sistemTalimati)},
	}

	session := model.StartChat()

	// Yapay zekaya mesajı gönderiyoruz
	resp, err := session.SendMessage(ctx, genai.Text(kullaniciMesaji))
	if err != nil {
		log.Fatal("Mesaj gönderilemedi:", err)
	}

	// 4. ADIM: FONKSİYON ÇAĞRISINI YAKALAMA VE İŞLEME
	for _, part := range resp.Candidates[0].Content.Parts {
		if funcCall, ok := part.(genai.FunctionCall); ok {
			fmt.Printf("Yapay Zeka bir fonksiyon çağırdı: %s\n", funcCall.Name)

			// Parametreleri Go struct'ına çeviriyoruz
			istekVerisi, _ := json.Marshal(funcCall.Args)
			var hesaplamaIstegi Hesaplamaİstegi
			json.Unmarshal(istekVerisi, &hesaplamaIstegi)

			// Backend fonksiyonumuzu çalıştırıyoruz
			sonuc := ikametUcretiHesapla(hesaplamaIstegi)
			fmt.Printf("Hesaplanan Sonuç: %+v\n\n", sonuc)

			// Sonucu JSON yapıp tekrar yapay zekaya gönderiyoruz ki kullanıcıya güzel bir cümle kursun
			sonucJSON, _ := json.Marshal(sonuc)
			resp, err = session.SendMessage(ctx, genai.FunctionResponse{
				Name: funcCall.Name,
				Response: map[string]any{
					"hesaplama_sonucu": string(sonucJSON),
				},
			})
			if err != nil {
				log.Fatal("Fonksiyon cevabı gönderilemedi:", err)
			}
		}
	}

	// Nihai cevabı yazdır
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			fmt.Println("ASİSTANIN CEVABI:\n", text)
		}
	}
}
