package bangumi

import (
	"log"
	"os"
	"testing"
)

func TestSearchSubject(t *testing.T) {
	b := New()
	info, err := b.SearchSubject("組長娘と世話係", "2022-07-07")
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Name:", info.Name)
	log.Println("Date:", info.Date)
	log.Println("Image:", info.Image)
	log.Println("NameCn:", info.NameCn)
	log.Println("Tags:", info.Tags)
	log.Println("Score:", info.Score)
	log.Println("Id:", info.Id)
}

func TestGetSubjectInfo(t *testing.T) {
	b := New()
	info, err := b.GetSubjectInfo(364450)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(info.Writer)
}

func TestBangumi_ListSubjectPerson(t *testing.T) {
	/*b := New()
	info, err := b.ListSubjectPerson(364450)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(info)*/
	f, _ := os.Stat("../")
	log.Println(f)
}
