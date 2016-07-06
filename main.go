package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ExampleFileURL = "http://forever.codeforamerica.org/fellowship-2015-tech-interview/Violations-2012.csv"
)

/*
	to be perfectly honest this did take me ~45 minutes, mostly do to never having used csv files so I had to look it up.

	I turned the exposed functions into an interface so that they could easily be turned into
	a RESTful JSON API in the future. Or really honestly reused in any way you see fit, although
	if you wanted to turn this into a micro service API I'd use https://github.com/go-kit/kit
*/
type Category interface {
	GetEarliest() violation
	GetLatest() violation
	TotalViolations() int
}

type category struct {
	name       string
	violations []violation
}

type violation struct {
	id            int
	inspectionId  int
	enteredDate   time.Time
	closedDate    time.Time
	violationType string
}

func main() {
	fileName, err := downloadFile(ExampleFileURL)
	if err != nil {
		log.Fatalf("was unable to download file to to error: %s", err)
	}

	categories, err := createDataSet(fileName)
	if err != nil {
		log.Fatalf("unable to generate the data set due to error: %s", err)
	}

	fmt.Println("this is the following data in the csv file")
	for _, category := range categories {
		// not sure how you wanted me
		fmt.Printf("category: %s \n", category.name)
		earliest := category.GetEarliest()
		fmt.Printf("earliest: violation_id: %d time_stamp %s \n", earliest.id, earliest.enteredDate)
		latest := category.GetLatest()
		fmt.Printf("latest: violation_id: %d time_stamp %s \n", latest.id, latest.enteredDate)
		fmt.Printf("total: %d \n", category.TotalViolations())
	}
}

// sort all of the violations as they come in
func (c *category) addViolation(v violation) {
	c.violations = append(c.violations, v)
	sort.Sort(c)
}

// publicly exposed functions
func (c *category) GetEarliest() violation {
	return c.violations[0]
}

func (c *category) GetLatest() violation {
	return c.violations[len(c.violations)-1]
}

func (c *category) TotalViolations() int {
	return len(c.violations)
}

// used by the sort interface
func (c *category) Len() int {
	return len(c.violations)
}

// used by the sort interface
func (c *category) Less(i, j int) bool {
	return c.violations[i].enteredDate.Before(c.violations[j].enteredDate)
}

// used by the sort interface
func (c *category) Swap(i, j int) {
	c.violations[i], c.violations[j] = c.violations[j], c.violations[i]
}

func NewViolation(values []string) violation {
	startDate, _ := time.Parse("2006-01-02 00:00:00", values[3])
	var closeDate time.Time = time.Time{}
	if values[4] != "" {
		closeDate, _ = time.Parse("2006-01-02 00:00:00", values[4])
	}

	id, _ := strconv.Atoi(values[0])
	inspectionId, _ := strconv.Atoi(values[1])

	return violation{
		id:            id,
		inspectionId:  inspectionId,
		enteredDate:   startDate,
		closedDate:    closeDate,
		violationType: values[5],
	}
}

func downloadFile(url string) (string, error) {
	tokens := strings.Split(url, "/")
	filename := tokens[len(tokens)-1]

	// check if file exists if it does remove it
	if _, err := os.Stat(filename); os.IsExist(err) {
		if err = os.Remove(filename); err != nil {
			return "", err
		}

	}
	// create the new file we'll be using
	newFile, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer newFile.Close()

	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// copy the downloaded file to the new file we made
	if _, err = io.Copy(newFile, response.Body); err != nil {
		return "", err
	}

	return filename, nil
}

func createDataSet(filename string) (map[string]*category, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(file)
	// by setting this to 0 it will expect the same items that the first line had
	reader.FieldsPerRecord = 0
	reader.TrimLeadingSpace = true

	categories := make(map[string]*category)

	// first line is junk so I'm ignoring it
	reader.Read()

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := line[2]
		value := NewViolation(line)
		if val, ok := categories[name]; ok {
			val.addViolation(value)
		} else {
			categories[name] = &category{name, []violation{value}}
		}
	}

	return categories, nil
}
