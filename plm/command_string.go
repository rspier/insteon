// Code generated by "stringer -type=Command -linecomment=true"; DO NOT EDIT.

package plm

import "strconv"

const (
	_Command_name_0 = "NAK"
	_Command_name_1 = "Std Msg ReceivedExt Msg ReceivedX10 Msg ReceivedAll Link CompleteButton Event ReportUser Reset DetectedLink Cleanup ReportLink Record RespLink Cleanup Status"
	_Command_name_2 = "Get InfoSend All LinkSend INSTEON MsgSend X10 MsgStart All LinkCancel All LinkSet Host CategoryResetSet ACK MsgGet First All LinkGet Next All LinkSet ConfigGet Sender All LinkLED OnLED OffManage All Link RecordSet NAK Msg ByteSet NAK Msg Two BytesRF SleepGet Config"
)

var (
	_Command_index_1 = [...]uint8{0, 16, 32, 48, 65, 84, 103, 122, 138, 157}
	_Command_index_2 = [...]uint16{0, 8, 21, 37, 49, 63, 78, 95, 100, 111, 129, 146, 156, 175, 181, 188, 210, 226, 247, 255, 265}
)

func (i Command) String() string {
	switch {
	case i == 21:
		return _Command_name_0
	case 80 <= i && i <= 88:
		i -= 80
		return _Command_name_1[_Command_index_1[i]:_Command_index_1[i+1]]
	case 96 <= i && i <= 115:
		i -= 96
		return _Command_name_2[_Command_index_2[i]:_Command_index_2[i+1]]
	default:
		return "Command(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
