// Code generated by "stringer -type InterState,InterStatus,PeerType,RIBState,SendState examples2_test.go examples_test.go halfpike.go halfpike_test.go"; DO NOT EDIT.

package halfpike

import "strconv"

const _InterState_name = "IStateUnknownIStateEnabledIStateDisabled"

var _InterState_index = [...]uint8{0, 13, 26, 40}

func (i InterState) String() string {
	if i < 0 || i >= InterState(len(_InterState_index)-1) {
		return "InterState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _InterState_name[_InterState_index[i]:_InterState_index[i+1]]
}

const _InterStatus_name = "IStatUnknownIStatUpIStatDown"

var _InterStatus_index = [...]uint8{0, 12, 19, 28}

func (i InterStatus) String() string {
	if i < 0 || i >= InterStatus(len(_InterStatus_index)-1) {
		return "InterStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _InterStatus_name[_InterStatus_index[i]:_InterStatus_index[i+1]]
}

const _PeerType_name = "PTUnknownPTExternalPTInternal"

var _PeerType_index = [...]uint8{0, 9, 19, 29}

func (i PeerType) String() string {
	if i >= PeerType(len(_PeerType_index)-1) {
		return "PeerType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _PeerType_name[_PeerType_index[i]:_PeerType_index[i+1]]
}

const (
	_RIBState_name_0 = "RSUnknown"
	_RIBState_name_1 = "RSCompleteRSInProgress"
)

var (
	_RIBState_index_1 = [...]uint8{0, 10, 22}
)

func (i RIBState) String() string {
	switch {
	case i == 0:
		return _RIBState_name_0
	case 2 <= i && i <= 3:
		i -= 2
		return _RIBState_name_1[_RIBState_index_1[i]:_RIBState_index_1[i+1]]
	default:
		return "RIBState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}

const _SendState_name = "RSSendUnknownRSSendSyncRSSendNotSyncRSSendNoAdvertise"

var _SendState_index = [...]uint8{0, 13, 23, 36, 53}

func (i SendState) String() string {
	if i >= SendState(len(_SendState_index)-1) {
		return "SendState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _SendState_name[_SendState_index[i]:_SendState_index[i+1]]
}
