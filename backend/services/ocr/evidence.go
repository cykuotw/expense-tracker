package ocr

// Historical evidence identifiers remain available to the operator cleanup tool.
const ImprovementPolicyVersion = "ocr-improvement-v1"

func EvidenceTemporaryKey(account, run string) string {
	return "temporary/" + account + "/" + run + ".json.gz"
}

func EvidencePrefix(account, run string) string {
	return "confirmed/" + account + "/" + run + "/"
}
