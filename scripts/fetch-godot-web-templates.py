# 從 Godot 官方匯出範本（zip 格式的 .tpz，約 1.3 GB）只下載網頁版：用 HTTP Range 讀 zip 目錄與指定項目（zipfile 會檢查 CRC32）。CI 用。
import io, sys, urllib.request, zipfile

class HTTPRangeFile(io.RawIOBase):
    def __init__(self, url):
        self.url, self.pos = url, 0
        req = urllib.request.Request(url, method="HEAD")
        with urllib.request.urlopen(req) as r:
            self.size = int(r.headers["Content-Length"])
            self.url = r.url  # 跟隨 GitHub 的轉址
    def seekable(self): return True
    def readable(self): return True
    def tell(self): return self.pos
    def seek(self, off, whence=0):
        self.pos = {0: off, 1: self.pos + off, 2: self.size + off}[whence]
        return self.pos
    def readinto(self, b):
        n = min(len(b), self.size - self.pos)
        if n <= 0:
            return 0
        req = urllib.request.Request(self.url, headers={"Range": f"bytes={self.pos}-{self.pos + n - 1}"})
        with urllib.request.urlopen(req) as r:
            data = r.read()
        b[:len(data)] = data
        self.pos += len(data)
        return len(data)

url, out, names = sys.argv[1], sys.argv[2], sys.argv[3:]
z = zipfile.ZipFile(io.BufferedReader(HTTPRangeFile(url), buffer_size=1 << 20))
for n in names:
    info = z.getinfo(n)
    print(n, info.file_size, "bytes", flush=True)
    with z.open(info) as src, open(f"{out}/{n.split('/')[-1]}", "wb") as dst:
        while chunk := src.read(1 << 20):
            dst.write(chunk)
print("完成")
