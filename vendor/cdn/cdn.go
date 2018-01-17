package cdn

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	//  "github.com/aws/aws-sdk-go/aws/awserr"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Worker definition of a worker instance
type Worker struct {
	ACL         string          // s3 acl for uploaded files - for our use either "public" or "private"
	Bucket      string          // s3 bucket to upload to
	Subfolder   string          // s3 subfolder destination (if needed)
	Svc         *s3.S3          // instance of s3 svc
	FileChannel chan string     // the channel to get file names from (upload todo list)
	Wg          *sync.WaitGroup // wait group - to signal when worker is finished
	SourceDir   string          // where source files are to be uploaded
	DestDir     string          // where to move uploaded files to (on local box)
	ID          int             // worker id number for debugging
}

// worker to get all files inside a directory (recursively)
func getFileList(searchDir string, fileChannel chan string, numWorkers int, wg *sync.WaitGroup) {
	defer wg.Done() // signal we are finished at end of function or return

	// sub function of how to recurse/walk the directory structure of searchDir
	_ = filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {

		// check if it's a file/directory (we just want files)
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close() // close file handle on return
		fi, err := file.Stat()
		if fi.IsDir() {
			return nil
		}

		fmt.Println("before:", path)
		path = strings.Replace(path, searchDir, "", 1)
		fmt.Println("path:", path)
		fileChannel <- path // add file to the work channel (queue)
		return nil
	})

	// add num_workers empty files on as termination signal to them
	for i := 0; i < numWorkers; i++ {
		fileChannel <- ""
	}
}

// upload function for workers
// uploads a given file to s3
func (worker *Worker) upload(file string) (string, error) {

	// s3 destination file path
	destfile := worker.Subfolder + file
	cdndir := strings.Replace(destfile, "/tmp", "", -1)
	worker.println("uploading to " + cdndir)

	// open and read file
	dir := strings.Replace(worker.SourceDir+file, "./tmp/tmp/", "./tmp/", -1)
	f, err := os.Open(dir) // worker.SourceDir + file
	if err != nil {
		return "Couldn't open file", err
	}
	defer f.Close()
	fileInfo, _ := f.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	f.Read(buffer)
	fileBytes := bytes.NewReader(buffer)

	params := &s3.PutObjectInput{
		Bucket: aws.String(worker.Bucket),
		Key:    aws.String(cdndir),
		Body:   fileBytes,
		ACL:    aws.String(worker.ACL),
	}

	// try the actual s3 upload
	resp, err := worker.Svc.PutObject(params)
	if err != nil {
		return "", err
	}
	return awsutil.StringValue(resp), nil
}

// doUploads function for workers
//
// reads from the file channel (queue),
// calls upload function for each,
// then moves uploaded files to worker.DestDir
func (worker *Worker) doUploads() {
	defer worker.Wg.Done() // notify parent when I complete
	worker.println("doUploads() started")

	// loop until I receive "" as a termination signal
	for {
		file := <-worker.FileChannel
		if file == "" {
			break
		}
		worker.println("File to upload: " + file)
		response, err := worker.upload(file)
		if err != nil {
			worker.println("error uploading " + file + ": " + response + " " + err.Error())
		} else {
			worker.println(response)
			// make destination directory if needed
			// filename := path.Base(file)
			// directory := strings.Replace(file, "/"+filename, "", 1)
			// os.MkdirAll(worker.DestDir+directory, 0775)
			// move file
			// os.Rename(worker.SourceDir+file, worker.DestDir+file)
			dir := strings.Replace(worker.SourceDir+file, "./tmp/tmp/", "./tmp/", -1)

			os.Remove(dir)
		}
	}
	worker.println("doUploads() finished")
}

// function to print out messages prefixed with worker-[id]
func (worker *Worker) println(message string) {
	fmt.Println("Worker-" + strconv.Itoa(worker.ID) + ": " + message)
}

// Upload allows upload all files to CDN
func Upload(bucket string, subfolder string, numWorkers int, region string, acl string, sourceDir string, destDir string) {
	fmt.Println("Using options:")
	fmt.Println("bucket:", bucket)
	fmt.Println("subfolder:", subfolder)
	fmt.Println("num_workers:", numWorkers)
	fmt.Println("region:", region)
	fmt.Println("acl:", acl)
	fmt.Println("sourceDir:", sourceDir)
	fmt.Println("destDir:", destDir)

	var wg sync.WaitGroup
	wg.Add(numWorkers + 1) // add 1 to account for the get_file_list thread!

	// file channel and thread to get the files
	fileChannel := make(chan string, 0)
	go getFileList(sourceDir, fileChannel, numWorkers, &wg)

	// set up s3 credentials from environment variables
	// these are shared to every worker
	creds := credentials.NewEnvCredentials()

	fmt.Println("Starting " + strconv.Itoa(numWorkers) + " workers...")

	// create the desired number of workers
	for i := 1; i <= numWorkers; i++ {
		// make a new worker
		sess := session.New(&aws.Config{Region: aws.String(region), Credentials: creds, LogLevel: aws.LogLevel(1)})
		svc := s3.New(sess)
		worker := &Worker{ACL: acl, Bucket: bucket, Subfolder: subfolder, Svc: svc, FileChannel: fileChannel, Wg: &wg, SourceDir: sourceDir, DestDir: destDir, ID: i}
		go worker.doUploads()
	}

	// wait for all workers to finish
	// (1x file worker and all uploader workers)
	wg.Wait()
}
