package main

type Attachment struct {
	FileName string `json:"File_Name"`
	FileType string `json:"File_Type"`
	FilePath string `json:"File_Path"`
}

type ReportResult struct {
	KodeEmiten  string       `json:"KodeEmiten"`
	Attachments []Attachment `json:"Attachments"`
}

type APIResponse struct {
	ResultCount int            `json:"ResultCount"`
	Results     []ReportResult `json:"Results"`
}

type CompanyResult struct {
	Code        string       `json:"code"`
	Attachments []Attachment `json:"attachments"`
}

type DownloadProgress struct {
	Code     string  `json:"code"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
	Running  bool    `json:"running"`
}

type AggregateProgress struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
