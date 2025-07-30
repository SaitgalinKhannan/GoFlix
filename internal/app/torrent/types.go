package torrent

type Torrent struct {
	InfoHash          string  `json:"infoHash"`
	Name              string  `json:"name"`
	Magnet            string  `json:"magnet"`
	Size              int64   `json:"size"`
	Done              bool    `json:"done"`
	DownloadedPercent float32 `json:"downloadedPercent"`
}
