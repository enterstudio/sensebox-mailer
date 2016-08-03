package main

import (
	"encoding/base64"
	"io"
	"net/mail"

	"gopkg.in/gomail.v2"
)

/*
{
  "template": "registration",
  "lang": "en",
  "recipient": {
	"address": "email@address.com",
	"name": "Philip J. Fry"
  },
  "payload": {
    "user": {
      "firstname": "Philip J.",
      "lastname": "Fry",
      "apikey": "<some valid apikey>"
    },
    "box": {
      "id": "<some valid senseBox id>",
      "sensors": [
        {
          "title": "<some title>",
          "type": "<some type>",
          "id": "<some valid senseBox sensor id>"
        },
        ...
      ]
    }
  },
  "attachment": {
    "filename": "senseBox.ino",
    "contents": "<file contents in base64>"
  }
}
*/

var addressParser = mail.AddressParser{}

type MailRequestAttachment struct {
	Filename string `json:"filename"`
	Contents string `json:"contents"`
}

type MailRequestDecodedAttachment struct {
	Filename string
	Contents []byte
}

type MailRequestEmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type MailRequest struct {
	Template          string                  `json:"template"`
	Language          string                  `json:"lang"`
	Recipient         MailRequestEmailAddress `json:"recipient"`
	Payload           map[string]interface{}  `json:"payload"`
	Attachment        *MailRequestAttachment  `json:"attachment,omitempty"`
	DecodedAttachment MailRequestDecodedAttachment
	BuiltTemplate     string
	EmailFrom         MailRequestEmailAddress
	Subject           string
}

func (request *MailRequest) validateAndParseRequest() error {
	// parse the Recipients address
	_, err := addressParser.Parse(request.Recipient.Address)
	if err != nil {
		return err
	}

	// decode the Attachment
	if request.Attachment != nil {
		data, err := base64.StdEncoding.DecodeString(request.Attachment.Contents)
		if err != nil {
			return err
		}
		request.DecodedAttachment = MailRequestDecodedAttachment{
			Filename: request.Attachment.Filename,
			Contents: data,
		}
	}

	// Fill FromAddress and SenderName
	senderName, err := getTranslation(request.Language, request.Template, "fromName")
	if err != nil {
		return err
	}
	request.EmailFrom = MailRequestEmailAddress{
		Address: request.Template + "@" + ConfigFromDomain,
		Name:    senderName,
	}

	// Fill in Subject
	subj, err := getTranslation(request.Language, request.Template, "subject")
	if err != nil {
		return err
	}
	request.Subject = subj

	// execute the template
	s, err := prepareMailBody(request.Template, request.Language, request.Payload)
	if err != nil {
		return err
	}
	request.BuiltTemplate = s

	return nil
}

func SendMail(req MailRequest) error {

	err := req.validateAndParseRequest()
	if err != nil {
		return err
	}

	m := gomail.NewMessage(gomail.SetCharset("UTF-8"))
	m.SetHeader("From", m.FormatAddress(req.EmailFrom.Address, req.EmailFrom.Name))
	m.SetHeader("To", m.FormatAddress(req.Recipient.Address, req.Recipient.Name))
	m.SetHeader("Subject", req.Subject)
	m.SetBody("text/html", req.BuiltTemplate)
	m.Attach(req.DecodedAttachment.Filename, gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(req.DecodedAttachment.Contents)
		return err
	}))

	d := gomail.NewDialer(ConfigSmtpServer, ConfigSmtpPort, ConfigSmtpUser, ConfigSmtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

//func (mailer *senseBoxMailerServer) startMailerDaemon() chan *gomail.Message {
//	ch := make(chan *gomail.Message)

//	go func() {
//		d := gomail.NewDialer("smtp.example.com", 587, "user", "123456")

//		var s gomail.SendCloser
//		var err error
//		open := false
//		for {
//			select {
//			case m, ok := <-ch:
//				if !ok {
//					return
//				}
//				if !open {
//					if s, err = d.Dial(); err != nil {
//						panic(err)
//					}
//					open = true
//				}
//				if err := gomail.Send(s, m); err != nil {
//					log.Print(err)
//				}
//			// Close the connection to the SMTP server if no email was sent in
//			// the last 30 seconds.
//			case <-time.After(30 * time.Second):
//				if open {
//					if err := s.Close(); err != nil {
//						panic(err)
//					}
//					open = false
//				}
//			}
//		}
//	}()

//	// Use the channel in your program to send emails.

//	// Close the channel to stop the mail daemon.
//	defer close(ch)
//	return ch
//}