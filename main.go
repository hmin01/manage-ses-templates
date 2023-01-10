package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	// AWS
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	// Env
	"github.com/joho/godotenv"
)

type Template struct {
	Name    string `json:"name"`
	Html    string `json:"html"`
	Subject string `json:"subject"`
}

func main() {
	ctx := context.TODO()
	// Create a client
	client := Configuration(ctx)

	// 플래그
	typePtr := flag.String("type", "get", "AWS SES 템플릿 관리 명령어\nex) create, delete, get, list, update")
	// 플래그 분석
	flag.Parse()
	// 플래그 확인
	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	if *typePtr == "get" {
		GetTemplate(ctx, client)
	} else if *typePtr == "list" {
		GetTemplates(ctx, client)
	} else if *typePtr == "create" {
		SetTemplate(ctx, client, true)
	} else if *typePtr == "delete" {
		DeleteTemplate(ctx, client)
	} else if *typePtr == "update" {
		SetTemplate(ctx, client, false)
	} else if *typePtr == "test" {
		SendEmail(ctx, client)
	} else {
		flag.Usage()
	}
}

func Configuration(ctx context.Context) *sesv2.Client {
	// 환경변수 불러오기
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("[ENV ERROR] %v", err)
	}
	// AWS Credentials values
	AWS_ACCESS_KEY := os.Getenv("AWS_ACCESS_KEY_ID")
	AWS_SECRET_KEY := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// Configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_ACCESS_KEY, AWS_SECRET_KEY, "")))
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	// Create a client
	return sesv2.NewFromConfig(cfg)
}

func DeleteTemplate(ctx context.Context, client *sesv2.Client) {
	// 삭제하고자 하는 템플릿 이름 입력
	var name string
	fmt.Print("삭제하고자 하는 템플릿 이름: ")
	fmt.Scanf("%s", &name)
	// 템플릿 삭제
	_, err := client.DeleteEmailTemplate(ctx, &sesv2.DeleteEmailTemplateInput{TemplateName: aws.String(name)})
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	// 삭제 완료 알림
	fmt.Println("템플릿 삭제 완료")
}

func GetTemplate(ctx context.Context, client *sesv2.Client) {
	// 찾고자 하는 템플릿 이름 입력
	var name string
	fmt.Print("찾고자 하는 템플릿 이름: ")
	fmt.Scanf("%s", &name)
	// 입력 받은 이름을 이용한 검색
	output, err := client.GetEmailTemplate(ctx, &sesv2.GetEmailTemplateInput{TemplateName: aws.String(name)})
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	// 검색 내용 출력
	fmt.Println("템플릿 제목: ", *output.TemplateContent.Subject)
	fmt.Println("템플릿 내용: ", *output.TemplateContent.Html)
}

func GetTemplates(ctx context.Context, client *sesv2.Client) {
	// Paginator 생성
	paginator := sesv2.NewListEmailTemplatesPaginator(client, &sesv2.ListEmailTemplatesInput{PageSize: aws.Int32(10)})
	// Recover
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("[ERROR] %v\n", r)
		}
	}()
	// 목록 객체 생성
	var list []string
	// 데이터 조회
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			log.Fatalf("[ERROR] Pagination error, %v\n", err)
		}
		// 데이터 추출
		for _, elem := range output.TemplatesMetadata {
			list = append(list, *elem.TemplateName)
		}
	}
	// 목록 출력
	if len(list) == 0 {
		fmt.Println("생성된 템플릿이 없습니다.")
	} else {
		fmt.Println("=-=-=- 조회 결과 -=-=-=")
		for _, elem := range list {
			fmt.Println(elem)
		}
	}
}

func SendEmail(ctx context.Context, client *sesv2.Client) {
	// 템플릿 이름
	var name string
	fmt.Print("메일을 보낼 템플릿 이름: ")
	fmt.Scanf("%s", &name)
	// 송신 메일 주소
	var target string
	fmt.Print("송신할 이메일 주소: ")
	fmt.Scanf("%s", &target)

	// 입력 객체 생성
	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateData: aws.String("{ \"name\": \"테스팅\" }"),
				TemplateName: aws.String(name),
			},
		},
		Destination: &types.Destination{
			ToAddresses: []string{target},
		},
		FromEmailAddress: aws.String("Plip <contact@plip.kr>"),
	}
	// 이메일 전송
	_, err := client.SendEmail(ctx, input)
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	// 결과 알림
	fmt.Println("이메일 전송 완료")
}

func SetTemplate(ctx context.Context, client *sesv2.Client, isCreate bool) {
	// 미리 생성한 템플릿 파일 이름 입력
	var filename string
	fmt.Print("미리 생성한 템플릿 설정 파일 이름: ")
	fmt.Scanf("%s", &filename)
	// 결과 객체 생성
	var result Template

	// 설정 파일 읽기
	data, err := ioutil.ReadFile("./templates/" + filename + ".json")
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	// JSON 변환
	json.Unmarshal(data, &result)

	// 템플릿(HTML) 파일 읽기
	html, err := ioutil.ReadFile("./templates/" + filename + ".html")
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	// 템플릿 내용 설정
	result.Html = string(html)

	if isCreate {
		// 입력 객체 생성
		input := &sesv2.CreateEmailTemplateInput{
			TemplateName: aws.String(result.Name),
			TemplateContent: &types.EmailTemplateContent{
				Subject: aws.String(result.Subject),
				Html:    aws.String(result.Html),
			},
		}
		// 템플릿 생성
		_, err = client.CreateEmailTemplate(ctx, input)
		if err != nil {
			log.Fatalf("[ERROR] %v", err)
		}
		// 생성 완료 알림
		fmt.Println("템플릿 생성 완료")
	} else {
		// 입력 객체 생성
		input := &sesv2.UpdateEmailTemplateInput{
			TemplateName: aws.String(result.Name),
			TemplateContent: &types.EmailTemplateContent{
				Subject: aws.String(result.Subject),
				Html:    aws.String(result.Html),
			},
		}
		// 템플릿 업데이트
		_, err = client.UpdateEmailTemplate(ctx, input)
		if err != nil {
			log.Fatalf("[ERROR] %v", err)
		}
		// 생성 완료 알림
		fmt.Println("템플릿 업데이트 완료")
	}
}
