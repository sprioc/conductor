package model

import "fmt"

type Ratio struct {
	Num int64  `bson:"-" json:"-"`
	Den int64  `bson:"-" json:"-"`
	Rep string `bson:"rep" json:"rep"`
}

func (r Ratio) Format() string {
	return fmt.Sprintf("%d", (r.Num / r.Den))
}

// NewRatio is a shortcut to make ratio types
func NewRatio(num, den int64, rep string) Ratio {
	return Ratio{Num: num, Den: den, Rep: rep}
}
