package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Sugi275/serless_metadeta-to-oracledb/loglib"
	fdk "github.com/fnproject/fdk-go"
	"github.com/google/uuid"
	_ "github.com/mattn/go-oci8"
)

const (
	envBucketName        = "OCI_BUCKETNAME"
	envSourceRegion      = "OCI_SOURCE_REGION"
	envTenancyName       = "OCI_TENANCY_NAME"
	envOracleUsername    = "ORACLE_USERNAME"
	envOraclePassword    = "ORACLE_PASSWORD"
	envOracleServiceName = "ORACLE_SERVICENAME"
	actionTypeCreate     = "com.oraclecloud.objectstorage.createobject"
	actionTypeUpdate     = "com.oraclecloud.objectstorage.updateobject"
	actionTypeDelete     = "com.oraclecloud.objectstorage.deleteobject"
)

// EventsInput EventsInput
type EventsInput struct {
	CloudEventsVersion string      `json:"cloudEventsVersion"`
	EventID            string      `json:"eventID"`
	EventType          string      `json:"eventType"`
	Source             string      `json:"source"`
	EventTypeVersion   string      `json:"eventTypeVersion"`
	EventTime          time.Time   `json:"eventTime"`
	SchemaURL          interface{} `json:"schemaURL"`
	ContentType        string      `json:"contentType"`
	Extensions         struct {
		CompartmentID string `json:"compartmentId"`
	} `json:"extensions"`
	Data struct {
		CompartmentID      string `json:"compartmentId"`
		CompartmentName    string `json:"compartmentName"`
		ResourceName       string `json:"resourceName"`
		ResourceID         string `json:"resourceId"`
		AvailabilityDomain string `json:"availabilityDomain"`
		FreeFormTags       struct {
			Department string `json:"Department"`
		} `json:"freeFormTags"`
		DefinedTags struct {
			Operations struct {
				CostCenter string `json:"CostCenter"`
			} `json:"Operations"`
		} `json:"definedTags"`
		AdditionalDetails struct {
			Namespace        string `json:"namespace"`
			PublicAccessType string `json:"publicAccessType"`
			ETag             string `json:"eTag"`
		} `json:"additionalDetails"`
	} `json:"data"`
}

//Image Image
type Image struct {
	ID         string
	ImageName  string
	Detail     string
	ImageURL   string
	Owner      string
	CreateDate time.Time
	Deleted    int
	Context    context.Context
}

func main() {
	fdk.Handle(fdk.HandlerFunc(fnMain))

	// ------- local development ---------
	// reader := os.Stdin
	// writer := os.Stdout
	// fnMain(context.TODO(), reader, writer)
}

func fnMain(ctx context.Context, in io.Reader, out io.Writer) {
	// Events から受け取るパラメータ
	input := &EventsInput{}
	json.NewDecoder(in).Decode(input)
	outputJSON, _ := json.Marshal(&input)
	fmt.Println(string(outputJSON))

	loglib.InitSugar()
	defer loglib.Sugar.Sync()

	// test
	fmt.Println("test print")
	loglib.Sugar.Infof("test info")
	err := fmt.Errorf("test error")
	loglib.Sugar.Error(err)
	fmt.Println("test print")

	imageConst, err := getImageConst(input.Data.ResourceName)
	if err != nil {
		loglib.Sugar.Error(err)
		return
	}

	err = saveImageMetadata(imageConst)
	if err != nil {
		loglib.Sugar.Error(err)
		return
	}
}

func getImageConst(imageName string) (Image, error) {
	// Generate certificate name
	const DateFormat = "20060102-1504"
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	imageURL, err := getImageURL(imageName)
	if err != nil {
		loglib.Sugar.Error(err)
		return Image{}, err
	}

	image := Image{
		ID:         uuid.New().String(),
		ImageName:  imageName,
		Detail:     "",
		ImageURL:   imageURL,
		Owner:      "",
		CreateDate: time.Now().In(jst),
		Deleted:    0, // 0:active 1:deleted
		Context:    context.Background(),
	}

	loglib.Sugar.Infof("Generated ImageConst. ID:" + image.ID + " ImageName:" + image.ImageName + " ImageURL:" + image.ImageURL + " CreateDate:" + image.CreateDate.Format(DateFormat))

	return image, nil
}

func getImageURL(imageName string) (string, error) {
	regionName, ok := os.LookupEnv(envSourceRegion)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envSourceRegion)
		loglib.Sugar.Error(err)
		return "", err
	}

	tenancyName, ok := os.LookupEnv(envTenancyName)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envTenancyName)
		loglib.Sugar.Error(err)
		return "", err
	}

	bucketName, ok := os.LookupEnv(envBucketName)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envBucketName)
		loglib.Sugar.Error(err)
		return "", err
	}

	url := "https://objectstorage." + regionName + ".oraclecloud.com/n/" + tenancyName + "/b/" + bucketName + "/o/" + imageName

	loglib.Sugar.Infof("Generated ImageURL:" + url)

	return url, nil
}

func saveImageMetadata(imageConst Image) error {
	dsn, err := getDSN()
	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	db, err := sql.Open("oci8", dsn)
	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	defer db.Close()

	err = insertMetadata(db, imageConst)
	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	return nil
}

func insertMetadata(db *sql.DB, imageConst Image) error {
	query := "INSERT INTO IMAGES (id, ImageName, Detail, ImageURL, UserName, CREATE_DATE, DELETED) " +
		"values (:1, :2, :3, :4, :5, :6, :7)"

	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	_, err := db.ExecContext(ctx, query, imageConst.ID, imageConst.ImageName, imageConst.Detail, imageConst.ImageURL, imageConst.Owner, imageConst.CreateDate, imageConst.Deleted)
	cancel()
	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	loglib.Sugar.Infof("Successful. Insert Metadata")

	return nil
}

func getDSN() (string, error) {
	oracleUsername, ok := os.LookupEnv(envOracleUsername)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envOracleUsername)
		loglib.Sugar.Error(err)
		return "", err
	}

	oraclePassword, ok := os.LookupEnv(envOraclePassword)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envOraclePassword)
		loglib.Sugar.Error(err)
		return "", err
	}

	oracleServiceName, ok := os.LookupEnv(envOracleServiceName)
	if !ok {
		err := fmt.Errorf("can not read environment variable %s", envOracleServiceName)
		loglib.Sugar.Error(err)
		return "", err
	}

	connect := oracleUsername + "/" + oraclePassword + "@" + oracleServiceName
	secretedConnect := oracleUsername + "/secret@" + oracleServiceName

	loglib.Sugar.Infof("Generated connect:" + secretedConnect)

	return connect, nil
}
