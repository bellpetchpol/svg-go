package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	// _ "github.com/heroku/x/hmetrics/onload"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/patrickmn/go-cache"
	"googlemaps.github.io/maps"

	firebase "firebase.google.com/go"
	_ "firebase.google.com/go/storage"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/api/option"
)

var Cache = cache.New(5*time.Minute, 5*time.Minute)

type requestBody struct {
	Action string `json:"Action"`
	Body   struct {
		NumberOfPosition string `json:"NumberOfPosition"`
		PlaceToFind      string `json:"PlaceToFind"`
		FieldsToFind     string `json:"FieldsToFind"`
		Message          string `json:"Message"`
	}
}

type allNumber []int

func setCache(key string, allNumber interface{}) bool {
	Cache.Set(key, allNumber, cache.NoExpiration)
	return true
}

func getCache(key string) (allNumber, bool) {
	var newAllNumber allNumber
	var found bool
	data, found := Cache.Get(key)
	if found {
		newAllNumber = data.(allNumber)
	}
	return newAllNumber, found
}

func main() {
	// port := os.Getenv("PORT")

	// if port == "" {
	// 	log.Fatal("$PORT must be set")
	// }

	router := gin.New()
	router.Use(gin.Logger())

	router.POST("/find_n_number", findNnumber)
	router.GET("/findRestaurantNearBangsue", findRestaurantNearBangsue)
	router.POST("/lineMessageAPI", lineMessageAPI)
	router.POST("/testFirebase", testFirebase)
	router.POST("/testFirebaseAddUser", testFirebaseAddUser)

	// router.Run(":" + port)
	router.Run(":8080")
}

func findNnumber(c *gin.Context) {
	var newRequestBody requestBody
	if err := c.BindJSON(&newRequestBody); err != nil {
		c.JSON(200, gin.H{"error": err.Error()})
		return
	}

	if newRequestBody.Action == "all" {
		if newRequestBody.Body.NumberOfPosition != "" {
			numOfPosition, err := strconv.Atoi(newRequestBody.Body.NumberOfPosition)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"error": err.Error()})
				return
			}
			// var numOfPosition int = newRequestBody.Body.NumberOfPosition
			var newAllNumber allNumber
			// var newAllNumberCache allNumberCache
			var newNumber int
			var cacheString = "all_number_" + strconv.Itoa(numOfPosition)
			newAllNumber, found := getCache(cacheString)
			if found {
				c.JSON(http.StatusOK, gin.H{
					"isFromCache": true,
					"data":        newAllNumber,
				})
			} else {

				for i := 0; i < numOfPosition; i++ {
					if i == 0 {
						newNumber = 3 // 5 = x + (i * 2) : 5 - 2 = x : 3
					} else {
						newNumber = newAllNumber[i-1] + (i * 2)
					}
					newAllNumber = append(newAllNumber[:i], newNumber)
				}

				setCache(cacheString, newAllNumber)
				c.JSON(http.StatusOK, gin.H{
					"isFromCache": false,
					"data":        newAllNumber,
				})
			}

			// json.NewEncoder(w).Encode(newAllNumber)
			// c.JSON(http.StatusOK, gin.H{"status": "you are logged in"})

		} else {
			// fmt.Fprintf(w, "Please specify numOfPosition")
			c.JSON(http.StatusOK, gin.H{
				"error": "Please specify numOfPosition",
			})
			return
		}

	} else {
		c.JSON(http.StatusOK, gin.H{
			"error": "Please specify valid action type",
		})
		return
	}
	// events = append(events, newEvent)
	// w.WriteHeader(http.StatusCreated)

}

type placesResult []maps.PlacesSearchResult

func setPlacesCache(key string, placesResult interface{}) bool {
	Cache.Set(key, placesResult, cache.NoExpiration)
	return true
}

func getPlacesCache(key string) (placesResult, bool) {
	var result []maps.PlacesSearchResult
	var found bool
	data, found := Cache.Get(key)
	if found {
		result := data.([]maps.PlacesSearchResult)
		return result, found
	}
	return result, found

}

func findRestaurantNearBangsue(c *gin.Context) {

	// var result string
	// var newPlacesResult placesResult
	var isFromCache bool
	// var latLng maps.LatLng
	var cacheKey string = "bangsue_all_nearby"

	cacheResult, found := getPlacesCache(cacheKey)
	if !found {

		ma, err := maps.NewClient(maps.WithAPIKey("API-KEY"))
		if err != nil {
			log.Fatalf("fatal error: %s", err)
		}
		var latLng = &maps.LatLng{
			Lat: 13.8496853,
			Lng: 100.5449568,
		}
		r := &maps.NearbySearchRequest{
			Location: latLng,
			Radius:   50000,
			Keyword:  "restaurant",
		}

		placeResponse, err := ma.NearbySearch(context.Background(), r)

		if err != nil {
			log.Fatalf("fatal error: %s", err)
		}

		// jsonResult, err := json.MarshalIndent(placeResponse.Results, "", "  ")
		// newPlacesResult = placesResult(jsonResult)
		setPlacesCache(cacheKey, placeResponse.Results)
		// result = string(jsonResult)
		isFromCache = false

		c.JSON(http.StatusOK, gin.H{
			"isFromCache": isFromCache,
			"data":        placeResponse.Results,
		})
	} else {
		// pretty.Fprintf(w, "from Cache")
		// result = string(cacheResult)
		isFromCache = true
		c.JSON(http.StatusOK, gin.H{
			"isFromCache": isFromCache,
			"data":        cacheResult,
		})
	}
	// fmt.Fprintf(w, placeResponse)
	// pretty.Fprintf(w, result)

}

type user struct {
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	TaxId      string `json:"taxId"`
	VerifyStep int64  `json:"verifyStep"`
	LineId     string `json:"lineId"`
}
type document struct {
	Id   string `json:"id"`
	User struct {
		Name       string `json:"name"`
		Surname    string `json:"surname"`
		TaxId      string `json:"taxId"`
		VerifyStep int64  `json:"verifyStep"`
		LineId     string `json:"lineId"`
	}
}
type documents []document

func lineMessageAPI(c *gin.Context) {
	// var newRequestBody requestBody
	// if err := c.BindJSON(&newRequestBody); err != nil {
	// 	c.JSON(http.StatusOK, gin.H{"error": err.Error()})
	// 	return
	// }
	bot, err := linebot.New("secret", "token")

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"bot new error": err.Error()})
		return
	}
	events, err := bot.ParseRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"parse request error": err.Error()})
		return
	}

	for _, event := range events {
		userID := event.Source.UserID
		// groupID := event.Source.GroupID
		// RoomID := event.Source.RoomID
		replyToken := event.ReplyToken
		if event.Type == linebot.EventTypeMessage {
			var messages []linebot.SendingMessage
			var replyMsg string
			// body := event.MarshalJSON
			// messages = append(messages, linebot.NewTextMessage("left clicked"))
			switch msg := event.Message.(type) {
			case *linebot.TextMessage:

				msgLength := len(msg.Text)
				// replyMsg = "msg.Text[0:5] : " + msg.Text[0:5]
				// messages = append(messages, linebot.NewTextMessage(replyMsg))
				// replyMsg = "msg.Text[msgLength-1:msgLength] : " + msg.Text[msgLength-1:msgLength]
				// messages = append(messages, linebot.NewTextMessage(replyMsg))
				if strings.ToLower(msg.Text[0:5]) == "taxid" && strings.ToLower(msg.Text[msgLength-1:msgLength]) == "#" {
					taxID := msg.Text[5 : msgLength-1]
					replyMsg = "getting taxid : " + taxID
					messages = append(messages, linebot.NewTextMessage(replyMsg))

					ctx := context.Background()
					opt := option.WithCredentialsFile("scg-candidate-firebase-adminsdk-40jmc-a4c792ce83.json")
					app, err := firebase.NewApp(context.Background(), nil, opt)
					if err != nil {
						// c.JSON(http.StatusOK, gin.H{"error initializing app: %v": err.Error()})
						replyMsg = "error initializing app: %v" + err.Error()
						messages = append(messages, linebot.NewTextMessage(replyMsg))
						return
					}

					client, err := app.Firestore(ctx)
					if err != nil {
						replyMsg = "error initializing Firestore: %v" + err.Error()
						messages = append(messages, linebot.NewTextMessage(replyMsg))
					}

					iter := client.Collection("users").Where("TaxId", "==", taxID).Documents(ctx)
					if iter == nil {
						replyMsg = "ขออภัยค่ะ รหัสบัตรประชาชน ไม่ตรงกับที่เรามี โปรดลองใหม่อีกครั้ง iter nil"
						messages = append(messages, linebot.NewTextMessage(replyMsg))
					}
					docs, err := iter.GetAll()

					if docs != nil {
						var documents documents
						for _, doc := range docs {
							// var user user
							var dct document
							mapstructure.Decode(doc.Data(), &dct.User)
							dct.Id = doc.Ref.ID
							dct.User.VerifyStep = 1
							dct.User.LineId = userID
							_, err := client.Collection("users").Doc(doc.Ref.ID).Set(ctx, dct.User)
							if err != nil {
								replyMsg = "ขออภัยค่ะ รหัสบัตรประชาชน ไม่ตรงกับที่เรามี โปรดลองใหม่อีกครั้ง setResult nil"
							} else {
								replyMsg = "ลงทะเบียนสำเร็จ ขอบคุณค่ะ\nคุณ" + dct.User.Name + " " + dct.User.Surname

							}
							messages = append(messages, linebot.NewTextMessage(replyMsg))
							documents = append(documents, dct)

						}
					} else {
						replyMsg = "ขออภัยค่ะ รหัสบัตรประชาชน ไม่ตรงกับที่เรามี โปรดลองใหม่อีกครั้ง setResult nil"
						messages = append(messages, linebot.NewTextMessage(replyMsg))
					}

				} else {
					replyMsg = msg.Text
					messages = append(messages, linebot.NewTextMessage(replyMsg))
				}
				// replyMsg = strconv.Itoa(msgLength)
				// messages = append(messages, linebot.NewTextMessage(replyMsg))

			}

			_, err := bot.ReplyMessage(replyToken, messages...).Do()
			if err != nil {
				// Do something when some bad happened
			}
		}
	}

}

func testFirebase(c *gin.Context) {
	var newRequestBody requestBody
	if err := c.BindJSON(&newRequestBody); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	opt := option.WithCredentialsFile("scg-candidate-firebase-adminsdk-40jmc-a4c792ce83.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error initializing app: %v": err.Error()})
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	iter := client.Collection("users").Where("taxId", "==", newRequestBody.Body.Message).Documents(ctx)
	docs, err := iter.GetAll()

	var documents documents
	for _, doc := range docs {
		// var user user
		var dct document
		mapstructure.Decode(doc.Data(), &dct.User)
		dct.Id = doc.Ref.ID
		dct.User.VerifyStep = 1
		dct.User.LineId = ""
		_, err := client.Collection("users").Doc(doc.Ref.ID).Set(ctx, dct.User)
		if err != nil {
		}
		documents = append(documents, dct)

	}
	c.JSON(http.StatusOK, gin.H{
		"data": documents,
	})
	// res, err := json.Marshal(documents)
	// if err != nil {

	// }
	// c.JSON(http.StatusOK, gin.H{
	// 	"data": res,
	// })
	// for {
	// 	// doc := iter.GetAll()
	// 	doc, err := iter.Next()
	// 	if err == iterator.Done {
	// 		break
	// 	}
	// 	// if err == iterator.Done {
	// 	// 		break
	// 	// }
	// 	if err != nil {
	// 		// log.Fatalf("Failed to iterate: %v", err)
	// 		c.JSON(http.StatusOK, gin.H{"Failed to iterate: %v": err.Error()})
	// 		return
	// 	}
	// 	// var users users
	// 	// doc.DataTo(&users)
	// 	// for _, user := range users {
	// 	// 	// replyMsg = "ขอบคุณสำหรับ TaxID:" + user.u.taxId
	// 	// }
	// 	// docResult, err := json.Unmarshal(doc.Data())

	// 	// if doc.Data() != nil {
	// 	// 	replyMsg = "ขอบคุณสำหรับ TaxID:" + taxId
	// 	// } else {
	// 	// 	replyMsg = "TaxID ไม่ตรงกับข้อมูลในระบบ"
	// 	// }
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"data": doc.Data(),
	// 	})
	// }

}

type fsUser struct {
	Name       string `firestore:"Name,omitempty"`
	Surname    string `firestore:"Surname,omitempty"`
	TaxID      string `firestore:"TaxId,omitempty"`
	VerifyStep int64  `firestore:"VerifyStep,omitempty"`
}

func testFirebaseAddUser(c *gin.Context) {
	var newRequestBody user
	if err := c.BindJSON(&newRequestBody); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	opt := option.WithCredentialsFile("scg-candidate-firebase-adminsdk-40jmc-a4c792ce83.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error initializing app: %v": err.Error()})
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	var newFsUser fsUser

	newFsUser.Name = newRequestBody.Name
	newFsUser.Surname = newRequestBody.Surname
	newFsUser.TaxID = newRequestBody.TaxId
	newFsUser.VerifyStep = 0

	docRef, _, err := client.Collection("users").Add(ctx, newFsUser)
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Insert success",
		"refID":   docRef.ID,
	})

}
