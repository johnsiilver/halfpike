// Code generated by "stringer -type=ItemType"; DO NOT EDIT.

package halfpike

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ItemUnknown-0]
	_ = x[ItemEOF-1]
	_ = x[ItemText-2]
	_ = x[ItemInt-3]
	_ = x[ItemFloat-4]
	_ = x[ItemEOL-5]
	_ = x[itemSpace-6]
}

const _ItemType_name = "ItemUnknownItemEOFItemTextItemIntItemFloatItemEOLitemSpace"

var _ItemType_index = [...]uint8{0, 11, 18, 26, 33, 42, 49, 58}

func (i ItemType) String() string {
	if i < 0 || i >= ItemType(len(_ItemType_index)-1) {
		return "ItemType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ItemType_name[_ItemType_index[i]:_ItemType_index[i+1]]
}
