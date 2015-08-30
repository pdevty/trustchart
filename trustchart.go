//  package main
//
//  import (
// 	  "fmt"
// 	  "github.com/pdevty/trustchart"
// 	  "io/ioutil"
// 	  "os"
//  )
//
//  func main() {
//
// 	  // params json format
// 	  // term 1y (1 year ) 2y 3y ...
// 	  //      1m (1 month) 2m 3m ...
// 	  //      1d (1 day  ) 2d 3d ...
// 	  // brand id,name from yahoo finance japan
// 	  params := `{
// 		"term":"1y",
// 		"brands":[
// 			{"id":"89311067","name":"jrevive"},
// 			{"id":"29311041","name":"ﾆｯｾｲ日経225"},
// 			{"id":"03316042","name":"健次"},
// 			{"id":"2931113C","name":"ﾆｯｾｲ外国株"}
// 		]
// 	  }`
//
// 	  // new
// 	  tc, err := trustchart.New(params)
// 	  if err != nil {
// 	    panic(err)
// 	  }
//
// 	  // return csv
// 	  fmt.Println(tc.Csv())
//
// 	  // return html
// 	  fmt.Println(tc.Html())
//
// 	  // create html file
// 	  ioutil.WriteFile("index.html",
// 	  	  []byte(tc.Html()), os.ModePerm)
//  }
package trustchart

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Map struct {
	Key   string
	Value string
}

type Maps []Map

type Data struct {
	Id   string
	Name string
	Data [][]string
}

type Brand struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Params struct {
	Term   string  `json:"term"`
	Brands []Brand `json:"brands"`
}

type Client struct {
	Header string
	Body   map[string]string
}

func getTerm(term string) (string, string) {
	to := time.Now()
	from := to
	num, _ := strconv.Atoi(string(term[0]))
	num *= -1
	switch string(term[1]) {
	case "d":
		from = to.AddDate(0, 0, num)
	case "m":
		from = to.AddDate(0, num, 0)
	case "y":
		from = to.AddDate(num, 0, 0)
	}
	return from.Format("20060102"), to.Format("20060102")
}

func getHtml() string {
	return `<html>
	<head>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/dygraph/1.2/dygraph-combined.min.js"></script>
	</head>
	<body>
		<div id="graphdiv"></div>
		<script>
			var csv = "";
			new Dygraph(document.getElementById("graphdiv"),csv);
		</script>
	</body>
</html>
`
}

func getData(b Brand) (Data, error) {
	url := "http://apl.morningstar.co.jp/webasp/shinkin/download.aspx?type=1&fnc=" + b.Id
	resp, err := http.Get(url)
	if err != nil {
		return Data{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Data{}, err
	}
	r := csv.NewReader(bytes.NewReader(body))
	records, err := r.ReadAll()
	if err != nil {
		return Data{}, err
	}
	return Data{
			Id:   b.Id,
			Name: b.Name,
			Data: records},
		nil
}

func New(params string) (*Client, error) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	p := Params{}
	json.Unmarshal([]byte(params), &p)
	from, to := getTerm(p.Term)
	datCh := make(chan Data)
	errCh := make(chan error)
	header := ""
	body := map[string]string{}
	for _, b := range p.Brands {
		go func(b Brand) {
			dat, err := getData(b)
			if err != nil {
				errCh <- err
			} else {
				datCh <- dat
			}
		}(b)
	}
	for i := 0; i < len(p.Brands); i++ {
		select {
		case err := <-errCh:
			return nil, err
		case dat := <-datCh:
			header += "," + dat.Name
			for _, d := range dat.Data {
				if from <= d[0] && to >= d[0] {
					body[d[0]] += "," + d[1]
				}
			}
		}
	}
	return &Client{
		Header: header,
		Body:   body,
	}, nil
}

func (m Maps) Len() int {
	return len(m)
}

func (m Maps) Less(i, j int) bool {
	return m[i].Key < m[j].Key
}

func (m Maps) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (c *Client) Csv() string {
	r := "Date" + c.Header + "\\n"
	ms := Maps{}
	for k, v := range c.Body {
		ms = append(ms, Map{Key: k, Value: v})
	}
	sort.Sort(ms)
	for _, v := range ms {
		r += v.Key + v.Value + "\\n"
	}
	return r
}

func (c *Client) Html() string {
	r := strings.Replace(getHtml(),
		"csv = \"\"",
		"csv = \""+c.Csv()+"\"", 1)
	return r
}
