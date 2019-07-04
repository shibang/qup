package main

import (
    "context"
    "fmt"
    "log"
    "net/url"
    "os"
    "path/filepath"
    "strings"

    "github.com/atotto/clipboard"
    "github.com/shibang/pb"
    "github.com/qiniu/api.v7/auth/qbox"
    "github.com/qiniu/api.v7/storage"
)

// UpHost 上传域名，不要改
const UpHost = "http://up.qbox.me"

var (
    Bucket string
    Domain string
    AK     string
    SK     string
)

func init() {
    Bucket = os.Getenv("Q_Bucket")
    Domain = os.Getenv("Q_Domain")
    AK = os.Getenv("Q_AK")
    SK = os.Getenv("Q_SK")
}

func main() {
    if len(os.Args) < 2 {
        fmt.Printf("Usage: %s <file> [new-name]\n", os.Args[0])
        return
    }

    if Bucket == "" || Domain == "" || AK == "" || SK == "" {
        fmt.Println("Please set Q_Bucket, Q_Domain, Q_AK, Q_SK environment variable")
    }

    fullName := os.Args[1]
    f, err := os.Open(fullName)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    stat, err := f.Stat()
    if err != nil {
        log.Fatal(err)
    }

    filename := filepath.Base(fullName)
    if len(os.Args) > 2 {
        filename = os.Args[2]
    }

    policy := storage.PutPolicy{
        Scope:   Bucket + ":" + filename,
        Expires: 360000,
    }
    mac := qbox.NewMac(AK, SK)
    token := policy.UploadToken(mac)

    storage.SetSettings(&storage.Settings{
        ChunkSize: 1024 * 1024 * 4,
    })

    conf := &storage.Config{
        UpHost: UpHost,
    }

    tmpl := `{{ green "⏳ Uploading:" }} {{counters . }} {{ bar . "[" "=" (cycle . ">" ) " " "]"}} {{percent .}} {{speed .}}`
    bar := pb.New64(stat.Size())
    bar.Set(pb.Bytes, true)
    bar.SetTemplateString(tmpl)
    if err = bar.Err(); err != nil {
        return
    }
    bar.Start()

    notifyCallback := func(blkIdx int, blkSize int, ret *storage.BlkputRet) {
        bar.Add64(int64(blkSize))
    }

    extra := &storage.RputExtra{
        TryTimes: 1000,
        Notify:   notifyCallback,
    }

    uploader := storage.NewResumeUploader(conf)
    if err := uploader.Put(context.Background(), nil, token, filename, f, stat.Size(), extra); err != nil {
        log.Fatal(err)
    }
    bar.Finish()

    if !strings.HasSuffix(Domain, "/") {
        Domain = Domain + "/"
    }
    dURL, err := url.Parse(Domain + filename)
    if err != nil {
        log.Fatal(err)
    }
    if dURL.Scheme == "" {
        dURL.Scheme = "http"
    }
    _ = clipboard.WriteAll(dURL.String())
    fmt.Println("✅ Download URL (already in clipboard): " + dURL.String())
}
