package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
)

type detGen struct {
	rng *rand.Rand
}

func newDetGen(seed uint64) *detGen {
	return &detGen{
		rng: rand.New(rand.NewPCG(seed, 100001)),
	}
}

func (g *detGen) intN(n int) int {
	return g.rng.IntN(n)
}

func (g *detGen) pick(items []string) string {
	return items[g.rng.IntN(len(items))]
}

func (g *detGen) chance(prob float64) bool {
	return g.rng.Float64() < prob
}

// computeUPN generates a valid 13-character UK DfE Unique Pupil Number (UPN)
// with an authentic modulo-23 check letter at position 1.
func computeUPN(laEstab string, year2Digit int, serial int) string {
	checkLetters := "ABCDEFGHJKLMNPQRTUVWXYZ"
	body := fmt.Sprintf("%s%02d%04d", laEstab, year2Digit%100, serial%10000)
	sum := 0
	for i := 0; i < len(body); i++ {
		digit := int(body[i] - '0')
		weight := i + 2
		sum += digit * weight
	}
	checkChar := checkLetters[sum%23]
	return fmt.Sprintf("%c%s", checkChar, body)
}

type encryptedVaultBlob struct {
	KMSKeyID          string
	WrappedDEK        []byte
	Nonce             []byte
	CiphertextPayload []byte
}

// encryptSafeguardingPayload performs deterministic AES-256-GCM envelope encryption
// so seeded safeguarding_notes rows contain genuine, decryptable AES-GCM payloads.
func encryptSafeguardingPayload(kmsKeyHandle string, noteID string, payload map[string]any) (*encryptedVaultBlob, error) {
	masterKey := sha256.Sum256([]byte("optimis-dsl-vault-master:" + kmsKeyHandle))
	dek := sha256.Sum256([]byte("optimis-dek:" + noteID))

	masterBlock, err := aes.NewCipher(masterKey[:])
	if err != nil {
		return nil, err
	}
	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return nil, err
	}
	wrapNonce := sha256.Sum256([]byte("wrap-nonce:" + noteID))
	wrappedDEK := masterGCM.Seal(nil, wrapNonce[:12], dek[:], []byte(noteID))

	dekBlock, err := aes.NewCipher(dek[:])
	if err != nil {
		return nil, err
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, err
	}
	payloadNonce := sha256.Sum256([]byte("payload-nonce:" + noteID))
	plainBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ciphertext := dekGCM.Seal(nil, payloadNonce[:12], plainBytes, []byte(noteID))

	return &encryptedVaultBlob{
		KMSKeyID:          kmsKeyHandle,
		WrappedDEK:        wrappedDEK,
		Nonce:             payloadNonce[:12],
		CiphertextPayload: ciphertext,
	}, nil
}

var ukFemaleFirstNames = []string{
	"Olivia", "Amelia", "Isla", "Ava", "Mia", "Ivy", "Lily", "Florence",
	"Freya", "Sienna", "Willow", "Phoebe", "Evie", "Sofia", "Grace", "Daisy",
	"Elsie", "Poppy", "Rosie", "Harper", "Maya", "Zara", "Fatima", "Aisha",
	"Niamh", "Chloe", "Eleanor", "Ruby", "Matilda", "Imogen", "Layla", "Erin",
	"Amira", "Anaya", "Hannah", "Jessica", "Lucy", "Charlotte", "Millie", "Esme",
	"Maisie", "Aria", "Orla", "Clara", "Martha", "Zainab", "Maryam", "Priya",
}

var ukMaleFirstNames = []string{
	"Noah", "Muhammad", "George", "Oliver", "Leo", "Arthur", "Oscar", "Theo",
	"Archie", "Jack", "Henry", "Charlie", "Freddie", "Harry", "Thomas", "Jude",
	"Alfie", "Jacob", "Lucas", "Isaac", "Liam", "Ryan", "Ethan", "Alexander",
	"Yusuf", "Ibrahim", "Hamza", "Rohan", "Aarav", "Daniel", "Samuel", "Joseph",
	"Benjamin", "Max", "Logan", "Reuben", "Caleb", "Toby", "Finley", "Harrison",
	"Sebastian", "Zachary", "Owen", "Dylan", "Gabriel", "Connor", "Callum", "Nathan",
}

var ukLastNames = []string{
	"Smith", "Jones", "Williams", "Taylor", "Brown", "Davies", "Evans", "Wilson",
	"Thomas", "Roberts", "Johnson", "Lewis", "Walker", "Robinson", "Wood", "Thompson",
	"White", "Watson", "Jackson", "Wright", "Green", "Harris", "Cooper", "King",
	"Lee", "Martin", "Clarke", "James", "Morgan", "Hughes", "Edwards", "Hill",
	"Moore", "Clark", "Harrison", "Scott", "Young", "Morris", "Hall", "Ward",
	"Turner", "Carter", "Phillips", "Mitchell", "Patel", "Khan", "Begum", "Ali",
	"Hussain", "Ahmed", "Shah", "Kaur", "Singh", "O'Connor", "Murphy", "Kelly",
	"Sullivan", "Walsh", "Kowalski", "Nowak", "Chen", "Sterling", "Grant", "Bennett",
	"Bailey", "Parker", "Price", "Cook", "Shaw", "Collins", "Bell", "Richards",
}

var ukStreets = []string{
	"High Street", "Station Road", "Church Lane", "Victoria Road", "Manor Way",
	"Park Avenue", "Queens Road", "London Road", "Kingsway", "The Green",
	"Mill Lane", "Woodland Close", "Meadow View", "Orchard Road", "Beech Grove",
	"St Giles Way", "Banbury Road", "Woodstock Road", "Cowley Road", "Iffley Road",
	"Headington Hill", "Botley Road", "Abingdon Road", "Marston Ferry Road",
}

var ukLocalities = []string{
	"Headington", "Summertown", "Jericho", "Cowley", "Iffley",
	"Marston", "Botley", "Wolvercote", "Rose Hill", "Littlemore",
}

var ukPostcodeOutwards = []string{"OX1", "OX2", "OX3", "OX4"}

var dfeEthnicities = []string{
	"WBRI", "WBRI", "WBRI", "WBRI", "WBRI", "WBRI",
	"WIRI", "WOTH", "MWBC", "MWBA", "INDN", "PKST",
	"BNGD", "CHNE", "BCRB", "BAFR", "OOTH", "NOBT",
}

var dfeEALLanguages = []string{
	"POL", "URD", "PAN", "ARA", "SPA", "POR", "BEN", "SOM", "ROU", "GUJ",
}

var autumn2026SchoolDays = []string{
	"2026-09-02", "2026-09-03", "2026-09-04",
	"2026-09-07", "2026-09-08", "2026-09-09", "2026-09-10", "2026-09-11",
	"2026-09-14", "2026-09-15", "2026-09-16", "2026-09-17", "2026-09-18",
	"2026-09-21", "2026-09-22", "2026-09-23", "2026-09-24", "2026-09-25",
	"2026-09-28", "2026-09-29",
}
