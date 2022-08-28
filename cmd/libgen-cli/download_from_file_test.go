package libgen_cli

import (
	"fmt"
	"reflect"
	"testing"
)

// Test Files Directory: test_data/
const TestData = "test_data/"

// Test_readInDOIs tests the readInDOIs function
func Test_readInDOIs(t *testing.T) {
	expected := []string{"10.1016/s0140-6736(19)32039-2", "10.1111/ijlh.13014"}

	filePath := fmt.Sprintf("%s/%s", TestData, "dois.json")
	results, err := readInDOIs(filePath)
	if err != nil {
		t.Error(err)
	}

	if reflect.DeepEqual(expected, results) == false {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

func Test_readInMd5s(t *testing.T) {
	expected := []string{"d41d8cd98f00b204e9800998ecf8427e", "d41d8cd98f00b204e9800998ecf8427e"}

	filePath := fmt.Sprintf("%s/%s", TestData, "md5s.json")
	results, err := readInMd5s(filePath)
	if err != nil {
		t.Error(err)
	}

	if reflect.DeepEqual(expected, results) == false {
		t.Errorf("Expected %v, got %v", expected, results)
	}
}

type addTest struct {
	arg1     string
	expected HashType
	actual   HashType
}

var identifyFileContentTests = []addTest{
	{"md5s.json", MD5, identifyFileContents(fmt.Sprintf("%s/%s", TestData, "md5s.json"))},
	{"md5s.json", DOI, identifyFileContents(fmt.Sprintf("%s/%s", TestData, "dois.json"))},
	{"hashes.json", MD5, identifyFileContents(fmt.Sprintf("%s/%s", TestData, "hashes.json"))},
}

func Test_identifyFileContents(t *testing.T) {

	for _, tst := range identifyFileContentTests {
		if tst.actual != tst.expected {
			t.Errorf("Expected %v, got %v", tst.actual, tst.expected)
		}
	}
}

func Example_readInMd5s() {
	filePath := fmt.Sprintf("%s/%s", TestData, "md5s.json")
	results, err := readInMd5s(filePath)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(results)
	// Output:
	// [d41d8cd98f00b204e9800998ecf8427e d41d8cd98f00b204e9800998ecf8427e]
}

func Example_readInDOIs() {
	filePath := fmt.Sprintf("%s/%s", TestData, "dois.json")
	results, err := readInDOIs(filePath)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(results)
	// Output:
	// [10.1016/s0140-6736(19)32039-2 10.1111/ijlh.13014]
}
