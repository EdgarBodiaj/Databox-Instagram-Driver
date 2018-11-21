package main

//IMAGE struct defines the data kept for each image
type IMAGE struct {
	StoreID string
	Size    DIMEN    `json:"dimensions"`
	DispURL string   `json:"display_url"`
	ImgID   string   `json:"id"`
	IsVid   bool     `json:"is_video"`
	Tags    []string `json:"tags"`
	TakenAt int64    `json:"taken_at_timestamp"`
	LIKE    struct {
		LikeCount int `json:"count"`
	} `json:"edge_media_preview_like"`
	CAPTION struct {
		Edges []EDGES `json:"edges"`
	} `json:"edge_media_to_caption"`
	COMMENT struct {
		CommentCount int `json:"count"`
	} `json:"edge_media_to_comment"`
}

//DIMEN struct holds the dimension of the image
type DIMEN struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type EDGES struct {
	Node NODES `json:"node"`
}

type NODES struct {
	Text string `json:"text"`
}
