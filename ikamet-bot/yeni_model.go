package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// Intent (Niyet), modelin soruyu sınıflandıracağı kategorileri temsil eder
type Intent struct {
	Tag       string
	Keywords  []string
	Responses []string
}

// QAModel, niyetleri barındıran ana yapımız
type QAModel struct {
	Intents []Intent
}

// Yeni bir Soru-Cevap modeli oluşturuyoruz
func NewQAModel() *QAModel {
	return &QAModel{
		Intents: []Intent{
			// --- ÖNCEKİ KATEGORİLER ---
			{
				Tag:      "selamlama",
				Keywords: []string{"merhaba", "selam", "hey", "günaydın", "iyi günler"},
				Responses: []string{
					"Merhaba! Size nasıl yardımcı olabilirim?",
					"Selam! Bugün sizin için ne yapabilirim?",
				},
			},
			{
				Tag:      "kargo_lojistik",
				Keywords: []string{"kargo", "navlun", "gönderim", "yurtdışı", "ne zaman", "teslimat", "takip"},
				Responses: []string{
					"Siparişleriniz genellikle 1-3 iş günü içinde kargoya verilir. Uluslararası lojistik partnerlerimizle Avrupa ve Amerika'ya güvenle teslimat sağlıyoruz.",
				},
			},
			// --- YENİ EKLENEN İKAMET KATEGORİLERİ ---
			{
				Tag:      "ikametgah_belgesi",
				Keywords: []string{"ikametgah", "ikamet", "belge", "nüfus", "kağıt", "çıktı", "nereden"},
				Responses: []string{
					"İkametgah (yerleşim yeri) belgenizi e-Devlet kapısı üzerinden barkodlu olarak saniyeler içinde oluşturup indirebilirsiniz.",
					"İkamet belgeniz için Nüfus Müdürlüklerine gitmenize gerek yok, e-Devlet üzerinden 'Yerleşim Yeri (İkametgah) Belgesi Sorgulama' ekranından alabilirsiniz.",
				},
			},
			{
				Tag:      "adres_degisikligi",
				Keywords: []string{"taşındım", "adres", "değişikliği", "taşıma", "bildirim", "yeni ev"},
				Responses: []string{
					"Yeni bir adrese taşındığınızda, adres değişikliği bildirimini 20 iş günü içinde Nüfus Müdürlüklerine veya şartları sağlıyorsanız doğrudan e-Devlet üzerinden yapmanız gerekmektedir.",
					"Boş bir konuta taşındıysanız adres değişikliğinizi e-Devlet'ten yapabilirsiniz. Ancak adreste başkası görünüyorsa Nüfus Müdürlüğüne gitmeniz gerekir.",
				},
			},
			{
				Tag:      "adres_onay_muvafakat",
				Keywords: []string{"onay", "muvafakat", "yetişkin", "limit", "deneme", "bloke", "birlikte oturma"},
				Responses: []string{
					"e-Devlet üzerinden bir başkasının (örneğin bir yetişkinin) yanına adres kaydı yapacaksanız, o kişinin size sistem üzerinden 'muvafakat' (onay) vermesi gerekir. İşlem sırasında SMS ile doğrulama yapılır.",
					"Adres onayı (muvafakat) işlemlerinde e-Devlet sisteminin güvenlik amacıyla belirlediği bir deneme limiti vardır. Bilgileri üst üste yanlış girerseniz sistem geçici olarak bloke olabilir, bu yüzden onay limitlerine dikkat etmelisiniz.",
				},
			},
		},
	}
}

// Ask, kullanıcıdan gelen metni analiz edip en mantıklı cevabı üretir
func (m *QAModel) Ask(question string) string {
	question = strings.ToLower(question)
	question = strings.ReplaceAll(question, "?", "")
	question = strings.ReplaceAll(question, ".", "")
	question = strings.ReplaceAll(question, "!", "")

	words := strings.Fields(question)

	bestMatchTag := ""
	maxScore := 0

	for _, intent := range m.Intents {
		score := 0
		for _, word := range words {
			for _, keyword := range intent.Keywords {
				if strings.Contains(word, keyword) {
					score++
				}
			}
		}

		if score > maxScore {
			maxScore = score
			bestMatchTag = intent.Tag
		}
	}

	if maxScore == 0 {
		return "Bu sorunun cevabından tam emin değilim. Acaba e-Devlet, ikametgah, adres değişikliği veya muvafakat gibi konulardan mı bahsediyorsunuz?"
	}

	rand.Seed(time.Now().UnixNano())
	for _, intent := range m.Intents {
		if intent.Tag == bestMatchTag {
			randomIndex := rand.Intn(len(intent.Responses))
			return intent.Responses[randomIndex]
		}
	}

	return "Bir hata oluştu."
}

func main() {
	model := NewQAModel()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🤖 İkamet ve Adres Bilgi Modeli Başlatıldı. (Çıkmak için 'kapat' yazın)")
	fmt.Println("------------------------------------------------------------------")

	for {
		fmt.Print("Siz: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "kapat" || strings.ToLower(input) == "çıkış" {
			fmt.Println("Model Kapatılıyor...")
			break
		}

		answer := model.Ask(input)
		fmt.Printf("Model: %s\n\n", answer)
	}
}
