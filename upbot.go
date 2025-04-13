package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cavaliergopher/grab/v3"
	"github.com/hashicorp/go-version"
)

var (
	pkgIndexUrl     = "https://dl.gitea.com/gitea/"
	pkgInnerInfoUrl = "https://www.sysinner.cn/ips/p1/pkg-info/entry?name=gitea"
	pkgInnerListUrl = "https://www.sysinner.cn/ips/p1/pkg/list?name=gitea&limit=100"
	/**
	  "kind": "PackList",
	  "items": [
	    {
	      "meta": {
	        "id": "gitea-1.23.5-1.linux.x64",
	        "name": "gitea",
	        "user": "sysadmin"
	      },
	      "version": {
	        "version": "1.23.5",
	        "release": "1",
	        "dist": "linux",
	        "arch": "x64"
	      },
	*/
	lastVersion = ""
)

func main() {

	{
		b, err := urlCall(pkgInnerInfoUrl)
		if err != nil {
			panic(err)
		}
		type respData struct {
			LastVersion string `json:"last_version"`
		}
		var rs respData
		if err = json.Unmarshal(b, &rs); err != nil {
			panic(err)
		}
		lastVersion = rs.LastVersion
	}
	lastSemver, err := version.NewVersion(lastVersion)
	if err != nil {
		panic(err)
	}

	body, err := urlCall(pkgIndexUrl)
	if err != nil {
		panic(err)
	}

	dom, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	vers := []*version.Version{}

	dom.Find("tr.file").Each(func(i int, s *goquery.Selection) {
		s.Find("span.name").Each(func(i2 int, s2 *goquery.Selection) {
			currSemver, err := version.NewVersion(s2.Text())
			if err != nil {
				return
			}
			if currSemver.LessThanOrEqual(lastSemver) {
				return
			}
			vers = append(vers, currSemver)
		})
	})

	sort.Slice(vers, func(i, j int) bool {
		return vers[i].Compare(vers[j]) < 0
	})

	for i, ver := range vers {
		vs := ver.Segments()
		if i+1 < len(vers) {
			v2s := vers[i+1].Segments()
			if v2s[0] == vs[0] && v2s[1] == vs[1] {
				println("skip", ver.String())
				continue
			}
		}

		if err := download(ver.String()); err != nil {
			return
		}
		for _, v := range [][]string{
			{"linux", "x64"},
		} {
			pkgname := fmt.Sprintf("gitea-%s-1.%s.%s.txz", ver.String(), v[0], v[1])
			if _, err := os.Stat(pkgname); err != nil && os.IsNotExist(err) {
				args := []string{
					"build",
					"-dist", v[0],
					"-arch", v[1],
					"--version", ver.String(),
				}
				fmt.Println("inpack start : inpack " + strings.Join(args, " "))
				out, err := exec.Command("inpack", args...).Output()
				if err != nil {
					fmt.Println("inpack build fail", err, string(out))
					continue
				}
				fmt.Println("inpack build done : ", pkgname)
			}

			args := []string{
				"push",
				"--repo",
				"bj1",
				"--pack_path",
				pkgname,
			}
			out, err := exec.Command("inpack", args...).Output()
			if err != nil {
				fmt.Println("inpack push fail", err, string(out))
				continue
			}

			fmt.Println("inpack push done : ", pkgname)
		}
	}
}

func download(ver string) error {

	localPath := fmt.Sprintf("deps/gitea-%s-linux-amd64", ver)
	reqUrl := fmt.Sprintf("%s%s/gitea-%s-linux-amd64.xz", pkgIndexUrl, ver, ver)

	_, err := os.Stat(localPath)
	if err == nil {
		return nil
	}

	_, err = os.Stat(localPath + ".xz")
	if err == nil {
		_, err = exec.Command("xz", "-d", localPath+".xz").Output()
		return err
	}

	// create client
	client := grab.NewClient()
	req, _ := grab.NewRequest(localPath+".xz", reqUrl)

	// start download
	fmt.Printf("Downloading %v...\n", req.URL())
	resp := client.Do(req)
	fmt.Printf("  %v\n", resp.HTTPResponse.Status)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf("  transferred %v / %v bytes (%.2f%%)\n",
				resp.BytesComplete(),
				resp.Size,
				100*resp.Progress())

		case <-resp.Done:
			break Loop
		}
	}

	if err := resp.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		return err
	}

	fmt.Printf("Download saved to ./%v \n", resp.Filename)

	_, err = exec.Command("xz", "-d", localPath+".xz").Output()

	return err
}

func urlCall(reqUrl string) ([]byte, error) {
	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
