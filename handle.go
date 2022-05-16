package function

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/g8rswimmer/go-twitter"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"errors"
	"strings"

)

var twitterToken = os.Getenv("TWITTER_TOKEN")
var redisHost = os.Getenv("REDIS_HOST") // This should include the port which is most of the time 6379
var redisPassword = os.Getenv("REDIS_PASSWORD")
var gameEventingEnabled = os.Getenv("GAME_EVENTING_ENABLED")
var sink = os.Getenv("GAME_EVENTING_BROKER_URI")
var cloudEventsEnabled bool = false

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}


type SessionScore struct{
	SessionId string
	Nickname string
	AccumulatedScore int
	LastLevel string
	Selected bool
}

type Leaderboard struct{
	Sessions []SessionScore
}

type Player struct{
	Nickname string
	AccumulatedScore int
}

type Winners struct{
	Players []Player
}

// Handle an HTTP Request.
func Handle(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	fmt.Println("Token?")
	fmt.Println(twitterToken)
	fmt.Println("Redis HOST?")
	fmt.Println(redisHost)

	var leaderboard Leaderboard
	resp, err := http.Post("http://get-leaderboard.default.svc.cluster.local", "application/json", nil)
	//Handle Error
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer resp.Body.Close()
	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(body, &leaderboard)
	if err != nil {
		log.Fatalln(err)
		return
	}

	query := "#bringbackthefunc"

	tweet := &twitter.Tweet{
		Authorizer: authorize{
			Token: twitterToken,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}
	fieldOpts := twitter.TweetFieldOptions{
		TweetFields: []twitter.TweetField{twitter.TweetFieldAuthorID, twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID, twitter.TweetFieldText},
	}
	searchOpts := twitter.TweetRecentSearchOptions{
		// look into the options for limiting timespan
	}

	recentSearch, err := tweet.RecentSearch(context.Background(), query, searchOpts, fieldOpts)

	//var winners Winners

	var tweetErr *twitter.TweetErrorResponse
	switch {
	case errors.As(err, &tweetErr):
		printTweetError(tweetErr)
	case err != nil:
		fmt.Println(err)
	default:
		topPlayersCounter := 0
		for _, lookup := range recentSearch.LookUps {
			// Check if the tweet contains the Nickname from the game and the same score
			if strings.Contains(lookup.Tweet.Text, leaderboard.Sessions[topPlayersCounter].Nickname) &&
				strings.Contains(lookup.Tweet.Text, string(leaderboard.Sessions[topPlayersCounter].AccumulatedScore)){

				fmt.Println("found winner tweet from: " + lookup.Tweet.AuthorID )
			}
			enc, err := json.MarshalIndent(lookup, "", "    ")
			if err != nil {
				panic(err)
			}
			fmt.Println(string(enc))
		}
		printRecentSearch(recentSearch)
		// For leaderboard.Sessions[0,1,2].Nickname
		//   Search for nickname in recentSearch, if so add to winners array
	}

	// Twitter client

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// Example implementation:
	fmt.Println("OK")       // Print "OK" to standard output (local logs)
	fmt.Fprintln(res, "OK") // Send "OK" back to the client
}

func printRecentSearch(recentSearch *twitter.TweetRecentSearch) {
	for _, lookup := range recentSearch.LookUps {
		enc, err := json.MarshalIndent(lookup, "", "    ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(enc))
	}
	//enc, err := json.MarshalIndent(recentSearch.Meta, "", "    ")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(string(enc))

}

func printTweetError(tweetErr *twitter.TweetErrorResponse) {
	enc, err := json.MarshalIndent(tweetErr, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(enc))
}