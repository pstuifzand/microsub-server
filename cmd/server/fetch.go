/*
   Microsub server
   Copyright (C) 2018  Peter Stuifzand

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pstuifzand/microsub-server/microsub"
	"willnorris.com/go/microformats"
)

type cacheItem struct {
	item    *microformats.Data
	created time.Time
}

var cache map[string]cacheItem

func init() {
	cache = make(map[string]cacheItem)
}

func Fetch2(fetchURL string) (*microformats.Data, error) {
	if !strings.HasPrefix(fetchURL, "http") {
		return nil, fmt.Errorf("error parsing %s as url", fetchURL)
	}

	u, err := url.Parse(fetchURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s as url: %s", fetchURL, err)
	}

	if data, e := cache[u.String()]; e {
		if data.created.After(time.Now().Add(time.Minute * -10)) {
			log.Printf("HIT %s - %s\n", u.String(), time.Now().Sub(data.created).String())
			return data.item, nil
		} else {
			log.Printf("EXPIRE %s\n", u.String())
			delete(cache, u.String())
		}
	} else {
		log.Printf("MISS %s\n", u.String())
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error while fetching %s: %s", u, err)
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return nil, fmt.Errorf("Content Type of %s = %s", fetchURL, resp.Header.Get("Content-Type"))
	}

	defer resp.Body.Close()
	data := microformats.Parse(resp.Body, u)
	cache[u.String()] = cacheItem{data, time.Now()}
	return data, nil
}

func Fetch(fetchURL string) []microsub.Item {
	result := []microsub.Item{}

	if !strings.HasPrefix(fetchURL, "http") {
		return result
	}

	u, err := url.Parse(fetchURL)
	if err != nil {
		log.Printf("error parsing %s as url: %s", fetchURL, err)
		return result
	}
	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("error while fetching %s: %s", u, err)
		return result
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		log.Printf("Content Type of %s = %s", fetchURL, resp.Header.Get("Content-Type"))
		return result
	}

	defer resp.Body.Close()
	data := microformats.Parse(resp.Body, u)
	jw := json.NewEncoder(os.Stdout)
	jw.SetIndent("", "    ")
	jw.Encode(data)

	author := microsub.Author{}

	for _, item := range data.Items {
		if item.Type[0] == "h-feed" {
			for _, child := range item.Children {
				previewItem := convertMfToItem(child)
				result = append(result, previewItem)
			}
		} else if item.Type[0] == "h-card" {
			mf := item
			author.Filled = true
			author.Type = "card"
			for prop, value := range mf.Properties {
				switch prop {
				case "url":
					author.URL = value[0].(string)
					break
				case "name":
					author.Name = value[0].(string)
					break
				case "photo":
					author.Photo = value[0].(string)
					break
				default:
					fmt.Printf("prop name not implemented for author: %s with value %#v\n", prop, value)
					break
				}
			}
		} else if item.Type[0] == "h-entry" {
			previewItem := convertMfToItem(item)
			result = append(result, previewItem)
		}
	}

	for i, item := range result {
		if !item.Author.Filled {
			result[i].Author = author
		}
	}

	return result
}

func convertMfToItem(mf *microformats.Microformat) microsub.Item {
	item := microsub.Item{}

	item.Type = mf.Type[0]

	for prop, value := range mf.Properties {
		switch prop {
		case "published":
			item.Published = value[0].(string)
			break
		case "url":
			item.URL = value[0].(string)
			break
		case "name":
			item.Name = value[0].(string)
			break
		case "latitude":
			item.Latitude = value[0].(string)
			break
		case "longitude":
			item.Longitude = value[0].(string)
			break
		case "like-of":
			for _, v := range value {
				item.LikeOf = append(item.LikeOf, v.(string))
			}
			break
		case "bookmark-of":
			for _, v := range value {
				item.BookmarkOf = append(item.BookmarkOf, v.(string))
			}
			break
		case "in-reply-to":
			for _, v := range value {
				item.InReplyTo = append(item.InReplyTo, v.(string))
			}
			break
		case "summary":
			if content, ok := value[0].(map[string]interface{}); ok {
				item.Content.HTML = content["html"].(string)
				item.Content.Text = content["value"].(string)
			} else if content, ok := value[0].(string); ok {
				item.Content.Text = content
			}
			break
		case "photo":
			for _, v := range value {
				item.Photo = append(item.Photo, v.(string))
			}
			break
		case "category":
			for _, v := range value {
				item.Category = append(item.Category, v.(string))
			}
			break
		case "content":
			if content, ok := value[0].(map[string]interface{}); ok {
				item.Content.HTML = content["html"].(string)
				item.Content.Text = content["value"].(string)
			} else if content, ok := value[0].(string); ok {
				item.Content.Text = content
			}
			break
		default:
			fmt.Printf("prop name not implemented: %s with value %#v\n", prop, value)
			break
		}
	}

	if item.Name == strings.TrimSpace(item.Content.Text) {
		item.Name = ""
	}

	// TODO: for like name is the field that is set
	if item.Content.HTML == "" && len(item.LikeOf) > 0 {
		item.Name = ""
	}

	fmt.Printf("%#v\n", item)
	return item
}
