package common

type SplitOption int

const (
	Equally SplitOption = iota
	Unequally

	YouHalf
	YouFull
	OtherHalf
	OtherFull
)

func (so SplitOption) String() string {
	mp := map[SplitOption]string{
		Equally:   "Equally",
		Unequally: "Unequally",
		YouHalf:   "You-Half",
		YouFull:   "You-Full",
		OtherHalf: "Other-Half",
		OtherFull: "Other-Full",
	}

	return mp[so]
}
