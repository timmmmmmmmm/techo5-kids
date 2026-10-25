package core

// Dutch clock phrases, the way children learn them in school: counting from the hour up to kwart
// over, then towards half; from half up to kwart voor, then towards the next hour. 07:20 is "tien
// voor half acht". Ported from the prototype's dutchTime.ts, and tested against the same fixture
// (testdata/dutch-time.json), so the two cannot drift apart.
//
// The device has no text-to-speech, so the clock plays a clip per minute of a twelve-hour day:
// cmd/kidsclips renders them, with macOS' say for free and Mistral's Voxtral for a nicer voice.

var hourWords = [12]string{"twaalf", "één", "twee", "drie", "vier", "vijf", "zes", "zeven", "acht", "negen", "tien", "elf"}

var numberWords = [15]string{
	"nul", "één", "twee", "drie", "vier", "vijf", "zes", "zeven", "acht", "negen",
	"tien", "elf", "twaalf", "dertien", "veertien",
}

// HourWord is the hour h spoken on the clock: noon and midnight are both "twaalf".
func HourWord(h int) string {
	return hourWords[((h%12)+12)%12]
}

// minutesWord is n minutes spoken, with its two forms for one.
func minutesWord(n int) string {
	if n == 1 {
		return "één minuut"
	}
	return numberWords[n]
}

// DutchTime is the exact phrase for h:m on a 24-hour clock, e.g. 07:20 → "tien voor half acht".
func DutchTime(h, m int) string {
	cur, next := HourWord(h), HourWord(h+1)
	switch {
	case m == 0:
		return cur + " uur"
	case m == 15:
		return "kwart over " + cur
	case m == 30:
		return "half " + next
	case m == 45:
		return "kwart voor " + next
	case m < 15:
		return minutesWord(m) + " over " + cur
	case m < 30:
		return minutesWord(30-m) + " voor half " + next
	case m < 45:
		return minutesWord(m-30) + " over half " + next
	}
	return minutesWord(60-m) + " voor " + next
}

// RoundedTime is the phrase rounded to five minutes, with "bijna" and "net … geweest" for the rest.
func RoundedTime(h, m int) string {
	total := h*60 + m
	r := (total + 2) / 5 * 5 // to the nearest five, without floats
	phrase := DutchTime((r/60)%24, r%60)
	if r == total {
		return phrase
	}
	if r > total {
		return "bijna " + phrase
	}
	return "net " + phrase + " geweest"
}

// TimeSentence is the sentence the clock says: "Het is tien voor half acht."
func TimeSentence(h, m int, rounded bool) string {
	phrase := DutchTime(h, m)
	if rounded {
		phrase = RoundedTime(h, m)
	}
	return "Het is " + phrase + "."
}

// NightSentence is a tap between bedtime and waketime: the time, and that it is still time to sleep —
// or, in the last half hour, that it is almost time to get up.
func NightSentence(h, m, minutesToWake int, rounded bool) string {
	t := TimeSentence(h, m, rounded)
	if minutesToWake <= 30 {
		return t + " Bijna tijd om op te staan."
	}
	return t + " Het is slaaptijd, blijf nog lekker liggen."
}
