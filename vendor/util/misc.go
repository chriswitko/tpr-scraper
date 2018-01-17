// Package util defines different utils
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"time"

	"github.com/araddon/dateparse"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

// ResponseStatus struct
type ResponseStatus struct {
	Code         int
	ErrorMessage string
}

// JSONStruct struct
type JSONStruct map[string]interface{}

// DefaultIntValue return int default
func DefaultIntValue(value int, replacement int) int {
	if value == 0 {
		return replacement
	}
	return value
}

// Contains func
func Contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// PrintFields func
func PrintFields(b interface{}, fieldName string) string {
	val := reflect.ValueOf(b)
	field, _ := val.Type().FieldByName(fieldName)
	return field.Tag.Get("structs")
}

func uniqueFileName(filename string) string {
	out := uuid.NewV4()
	extension := filepath.Ext(filename)
	if extension == "" {
		extension = ".jpg"
	}
	return out.String() + extension
}

// FilesExist func
func FilesExist(files ...string) error {
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("unable to stat %s: %v", f, err)
		}
	}
	return nil
}

// GetFileLink func
func GetFileLink(filename string, bucket string) string {
	url := "https://s3.amazonaws.com/%s/%s/%s"
	url = fmt.Sprintf(url, GetEnv("AWS_CUSTOM_BUCKET", ""), bucket, filename)
	return url
}

// RemoveUploadedFiles func
func RemoveUploadedFiles(folder string, files []string) ([]string, error) {
	cwd, _ := os.Getwd()
	// Removed uploaded files from local path
	for _, file := range files {
		os.Remove(filepath.Join(cwd, folder, file))
	}
	return files, nil
}

// Weekday func
func Weekday(w int) int {
	if w == 0 {
		return 7
	}
	return w
}

// CalculateNextAt func
func CalculateNextAt(user map[string]interface{}) (time.Time, error) { // (days []int, hours []string, timezone string) (time.Time, error) {
	timezone := user["timezone"].(string)
	days := user["days"].([]int)
	hours := user["hours"].([]string)
	isUnsubscribed := user["is_unsubscribed"].(bool)

	if !isUnsubscribed {
		return time.Time{}, nil
	}

	if timezone == "" {
		timezone = "Europe/London"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	time.Local = loc
	start, err := dateparse.ParseLocal(time.Now().UTC().String())
	end := start.Add(time.Hour * 24 * 7)

	if err != nil {
		return time.Time{}, err
	}
	var schedule []time.Time

	for d := start; d.Unix() <= end.Unix(); d = d.AddDate(0, 0, 1) {
		wd := Weekday(int(d.Weekday()))
		if ContainsInt(days, wd) {
			for _, hour := range hours {
				sn := fmt.Sprintf("%d-%02d-%02d %s", d.Year(), int(d.Month()), d.Day(), hour)
				currentTime, err := dateparse.ParseLocal(sn)
				if err != nil {
					return time.Time{}, err
				}
				if currentTime.Unix() >= start.Unix() {
					schedule = append(schedule, currentTime)
				}
			}
		}
	}

	sort.Slice(schedule, func(i, j int) bool {
		return schedule[i].Before(schedule[j])
	})

	if len(schedule) > 0 {
		return schedule[0], nil
	}
	return time.Time{}, nil
}

// ContainsInt func
func ContainsInt(slice []int, item int) bool {
	set := make(map[int]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// UploadFiles func
func UploadFiles(c *gin.Context, folder string, data JSONStruct, files []*multipart.FileHeader) ([]string, error) {
	var output []string
	for _, file := range files {
		tmpFileName := uniqueFileName(file.Filename)
		savedPath := filepath.Join(folder, tmpFileName)
		output = append(output, tmpFileName)
		err := c.SaveUploadedFile(file, savedPath)
		if err != nil {
			return nil, err
		}
	}
	return output, nil
}

// UploadToS3 func
func UploadToS3(folder string, uploadFolder string, files []string) ([]string, error) {
	var output []string
	cwd, _ := os.Getwd()
	s3Region := GetEnv("AWS_REGION", "")
	s3Bucket := GetEnv("AWS_CUSTOM_BUCKET", "")
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(s3Region)})
	if err != nil {
		return output, err
	}
	for _, file := range files {
		savedPath := filepath.Join(cwd, folder, file)
		// data[tmpFileName] = tmpFileName
		output = append(output, file)

		// Upload
		_, err = AddFileToS3(s, folder, uploadFolder, file, savedPath, s3Bucket)
		if err != nil {
			return output, err
		}
	}
	return output, nil
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, folder string, uploadFolder string, fileName string, fileDir string, s3Bucket string) (*s3.PutObjectOutput, error) {
	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		return &s3.PutObjectOutput{}, err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	location := uploadFolder + "/" + fileName

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	output, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(s3Bucket),
		Key:                  aws.String(location),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	return output, err
}

// MergeMaps manage global options
func MergeMaps(a JSONStruct, b JSONStruct) {
	for k, v := range b {
		a[k] = v
	}
}

// FormatErrorResponse func
func FormatErrorResponse(c *gin.Context, err string, code int) (int, JSONStruct) {
	return FormatResponse(c, JSONStruct{}, ResponseStatus{
		Code:         code,
		ErrorMessage: err,
	})
}

// FormatResponse func
func FormatResponse(c *gin.Context, body JSONStruct, status ResponseStatus) (int, JSONStruct) {
	re := regexp.MustCompile("(v[0-9]{1,2})")
	metaHeader := JSONStruct{
		"status":      status.Code,
		"request_id":  c.Writer.Header().Get("Request-Id"),
		"api_version": re.FindString(c.Request.RequestURI),
	}

	if status.ErrorMessage != "" {
		errorHeader := JSONStruct{
			"error_detail": status.ErrorMessage,
		}
		MergeMaps(metaHeader, errorHeader)
	}

	eq := reflect.DeepEqual(body, JSONStruct{})
	if eq {
		return status.Code, JSONStruct{
			"meta": metaHeader,
		}
	}
	return status.Code, JSONStruct{
		"meta":     metaHeader,
		"response": body,
	}
}

// ConvertModelToStruct func
func ConvertModelToStruct(any interface{}, output interface{}) interface{} {
	b, _ := json.Marshal(any)

	json.Unmarshal(b, &output)
	return output
}

// SignToken generate token
func SignToken(user map[string]interface{}, sub string) string {
	exp := map[string]time.Duration{
		"auth":        0,
		"unsubscribe": 0,
		"subscribe":   time.Minute * 30,
		"app":         0,
	}

	publicKey := []byte(GetEnv("PUBLIC_KEY", ""))

	payload := JSONStruct{
		"sub": sub,
		"iat": time.Now().Unix(),
		"uid": user["ID"], //user.ID.Hex(),
	}

	if exp[sub] == 0 {
		payload["exp"] = exp[sub]
	} else {
		payload["exp"] = time.Now().Add(exp[sub]).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(payload))

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(publicKey)
	return tokenString
}
