// Copyright (c) 2013 ActiveState Software Inc. All rights reserved.

// TODO:
//  * repeat all the tests with Poll:true

package tail

import (
	"./watch"
	_ "fmt"
	"github.com/ActiveState/tail/ratelimiter"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func init() {
	// Clear the temporary test directory
	err := os.RemoveAll(".test")
	if err != nil {
		panic(err)
	}
}

func TestMustExist(t *testing.T) {
	tail, err := TailFile("/no/such/file", Config{Follow: true, MustExist: true})
	if err == nil {
		t.Error("MustExist:true is violated")
		tail.Stop()
	}
	tail, err = TailFile("/no/such/file", Config{Follow: true, MustExist: false})
	if err != nil {
		t.Error("MustExist:false is violated")
	}
	tail.Stop()
	_, err = TailFile("README.md", Config{Follow: true, MustExist: true})
	if err != nil {
		t.Error("MustExist:true on an existing file is violated")
	}
	tail.Stop()
	Cleanup()
}

func TestStop(t *testing.T) {
	tail, err := TailFile("_no_such_file", Config{Follow: true, MustExist: false})
	if err != nil {
		t.Error("MustExist:false is violated")
	}
	if tail.Stop() != nil {
		t.Error("Should be stoped successfully")
	}
	Cleanup()
}

func TestMaxLineSize(_t *testing.T) {
	t := NewTailTest("maxlinesize", _t)
	t.CreateFile("test.txt", "hello\nworld\nfin\nhe")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: nil, MaxLineSize: 3})
	go t.VerifyTailOutput(tail, []string{"hel", "lo", "wor", "ld", "fin", "he"},
		[]int64{0, 3, 6, 9, 12, 16}, []uint64{inode, inode, inode, inode, inode, inode})

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	tail.Stop()
	Cleanup()
}

func TestOver4096ByteLine(_t *testing.T) {
	t := NewTailTest("Over4096ByteLine", _t)
	testString := strings.Repeat("a", 4097)
	t.CreateFile("test.txt", "test\n"+testString+"\nhello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: nil})
	go t.VerifyTailOutput(tail, []string{"test", testString, "hello", "world"},
		[]int64{0, 5, 4103, 4109}, []uint64{inode, inode, inode, inode})

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	tail.Stop()
	Cleanup()
}
func TestOver4096ByteLineWithSetMaxLineSize(_t *testing.T) {
	t := NewTailTest("Over4096ByteLineMaxLineSize", _t)
	testString := strings.Repeat("a", 4097)
	t.CreateFile("test.txt", "test\n"+testString+"\nhello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: nil, MaxLineSize: 4097})
	go t.VerifyTailOutput(tail, []string{"test", testString, "hello", "world"},
		[]int64{0, 5, 4103, 4109}, []uint64{inode, inode, inode, inode})

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	// tail.Stop()
	Cleanup()
}

func TestLocationFull(_t *testing.T) {
	t := NewTailTest("location-full", _t)
	t.CreateFile("test.txt", "hello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: nil})
	go t.VerifyTailOutput(tail, []string{"hello", "world"},
		[]int64{0, 6}, []uint64{inode, inode})

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	tail.Stop()
	Cleanup()
}

func TestLocationFullDontFollow(_t *testing.T) {
	t := NewTailTest("location-full-dontfollow", _t)
	t.CreateFile("test.txt", "hello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: false, Location: nil})
	go t.VerifyTailOutput(tail, []string{"hello", "world"},
		[]int64{0, 6}, []uint64{inode, inode})

	// Add more data only after reasonable delay.
	<-time.After(100 * time.Millisecond)
	t.AppendFile("test.txt", "more\ndata\n")
	<-time.After(100 * time.Millisecond)

	tail.Stop()
	Cleanup()
}

func TestLocationEnd(_t *testing.T) {
	t := NewTailTest("location-end", _t)
	t.CreateFile("test.txt", "hello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: &SeekInfo{0, os.SEEK_END}})
	go t.VerifyTailOutput(tail, []string{"more", "data"},
		[]int64{12, 17}, []uint64{inode, inode})

	<-time.After(100 * time.Millisecond)
	t.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	tail.Stop()
	Cleanup()
}

func TestLocationMiddle(_t *testing.T) {
	// Test reading from middle.
	t := NewTailTest("location-end", _t)
	t.CreateFile("test.txt", "hello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail("test.txt", Config{Follow: true, Location: &SeekInfo{-6, os.SEEK_END}})
	go t.VerifyTailOutput(tail, []string{"world", "more", "data"},
		[]int64{6, 12, 17}, []uint64{inode, inode, inode})

	<-time.After(100 * time.Millisecond)
	t.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")
	tail.Stop()
	Cleanup()
}

func _TestReOpen(_t *testing.T, poll bool) {
	var name string
	var delay time.Duration
	if poll {
		name = "reopen-polling"
		delay = 300 * time.Millisecond // account for POLL_DURATION
	} else {
		name = "reopen-inotify"
		delay = 100 * time.Millisecond
	}
	t := NewTailTest(name, _t)
	t.CreateFile("test.txt", "hello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: true, Poll: poll})

	if !poll {
		// deletion must trigger reopen
		<-time.After(delay)
		t.RemoveFile("test.txt")
		<-time.After(delay)
		t.CreateFile("test.txt", "more\ndata\n")
		inode_1 := t.FileInode("test.txt")

		// rename must trigger reopen
		<-time.After(delay)
		t.RenameFile("test.txt", "test.txt.rotated")
		<-time.After(delay)
		t.CreateFile("test.txt", "endofworld")
		inode_2 := t.FileInode("test.txt")

		go t.VerifyTailOutput(tail, []string{"hello", "world", "more", "data", "endofworld"},
			[]int64{0, 6, 0, 5, 0}, []uint64{inode, inode, inode_1, inode_1, inode_2})
	} else {
		// FIXME verify inode when poll
		go t.VerifyTailOutput(tail, []string{"hello", "world", "more", "data", "endofworld"},
			[]int64{0, 6, 0, 5, 0}, []uint64{inode, inode})

		// deletion must trigger reopen
		<-time.After(delay)
		t.RemoveFile("test.txt")
		<-time.After(delay)
		t.CreateFile("test.txt", "more\ndata\n")

		// rename must trigger reopen
		<-time.After(delay)
		t.RenameFile("test.txt", "test.txt.rotated")
		<-time.After(delay)
		t.CreateFile("test.txt", "endofworld")
	}

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(delay)
	t.RemoveFile("test.txt")
	<-time.After(delay)

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	// tail.Stop()
	Cleanup()
}

// The use of polling file watcher could affect file rotation
// (detected via renames), so test these explicitly.

func TestReOpenInotify(_t *testing.T) {
	_TestReOpen(_t, false)
}

func TestReOpenPolling(_t *testing.T) {
	_TestReOpen(_t, true)
}

func _TestReSeek(_t *testing.T, poll bool) {
	var name string
	if poll {
		name = "reseek-polling"
	} else {
		name = "reseek-inotify"
	}
	t := NewTailTest(name, _t)
	t.CreateFile("test.txt", "a really long string goes here\nhello\nworld\n")
	inode := t.FileInode("test.txt")
	tail := t.StartTail(
		"test.txt",
		Config{Follow: true, ReOpen: false, Poll: poll})

	go t.VerifyTailOutput(tail, []string{
		"a really long string goes here", "hello", "world", "h311o", "w0r1d", "endofworld"},
		[]int64{0, 31, 37, 0, 6, 12}, []uint64{inode, inode, inode, inode, inode, inode})

	// truncate now
	<-time.After(100 * time.Millisecond)
	t.TruncateFile("test.txt", "h311o\nw0r1d\nendofworld\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")

	// Do not bother with stopping as it could kill the tomb during
	// the reading of data written above. Timings can vary based on
	// test environment.
	// tail.Stop()
	Cleanup()
}

// The use of polling file watcher could affect file rotation
// (detected via renames), so test these explicitly.

func TestReSeekInotify(_t *testing.T) {
	_TestReSeek(_t, false)
}

func TestReSeekPolling(_t *testing.T) {
	_TestReSeek(_t, true)
}

func TestRateLimiting(_t *testing.T) {
	t := NewTailTest("rate-limiting", _t)
	t.CreateFile("test.txt", "hello\nworld\nagain\nextra\n")
	inode := t.FileInode("test.txt")
	config := Config{
		Follow:      true,
		RateLimiter: ratelimiter.NewLeakyBucket(2, time.Second)}
	leakybucketFull := "Too much log activity; waiting a second before resuming tailing"
	tail := t.StartTail("test.txt", config)

	// TODO: also verify that tail resumes after the cooloff period.
	go t.VerifyTailOutput(tail, []string{
		"hello", "world", "again",
		leakybucketFull,
		"more", "data",
		leakybucketFull}, []int64{0, 6, 12, PositionNone, 24, 29, PositionNone},
		[]uint64{inode, inode, inode, inode, inode, inode, inode})

	// Add more data only after reasonable delay.
	<-time.After(1200 * time.Millisecond)
	t.AppendFile("test.txt", "more\ndata\n")

	// Delete after a reasonable delay, to give tail sufficient time
	// to read all lines.
	<-time.After(100 * time.Millisecond)
	t.RemoveFile("test.txt")

	// tail.Stop()
	Cleanup()
}

func TestTell(_t *testing.T) {
	t := NewTailTest("tell-position", _t)
	t.CreateFile("test.txt", "hello\nworld\nagain\nmore\n")
	config := Config{
		Follow:   false,
		Location: &SeekInfo{0, os.SEEK_SET}}
	tail := t.StartTail("test.txt", config)
	// read noe line
	<-tail.Lines
	offset, err := tail.Tell()
	if err != nil {
		t.Errorf("Tell return error: %s", err.Error())
	}
	tail.Done()
	// tail.close()

	config = Config{
		Follow:   false,
		Location: &SeekInfo{offset, os.SEEK_SET}}
	tail = t.StartTail("test.txt", config)
	for l := range tail.Lines {
		// it may readed one line in the chan(tail.Lines),
		// so it may lost one line.
		if l.Text != "world" && l.Text != "again" {
			t.Fatalf("mismatch; expected world or again, but got %s",
				l.Text)
		}
		break
	}
	t.RemoveFile("test.txt")
	tail.Done()
	Cleanup()
}

// Test library

type TailTest struct {
	Name string
	path string
	*testing.T
}

func NewTailTest(name string, t *testing.T) TailTest {
	tt := TailTest{name, ".test/" + name, t}
	err := os.MkdirAll(tt.path, os.ModeTemporary|0700)
	if err != nil {
		tt.Fatal(err)
	}

	// Use a smaller poll duration for faster test runs. Keep it below
	// 100ms (which value is used as common delays for tests)
	watch.POLL_DURATION = 5 * time.Millisecond

	return tt
}

func (t TailTest) CreateFile(name string, contents string) {
	err := ioutil.WriteFile(t.path+"/"+name, []byte(contents), 0600)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) RemoveFile(name string) {
	err := os.Remove(t.path + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) RenameFile(oldname string, newname string) {
	oldname = t.path + "/" + oldname
	newname = t.path + "/" + newname
	err := os.Rename(oldname, newname)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) AppendFile(name string, contents string) {
	f, err := os.OpenFile(t.path+"/"+name, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(contents)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) TruncateFile(name string, contents string) {
	f, err := os.OpenFile(t.path+"/"+name, os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(contents)
	if err != nil {
		t.Fatal(err)
	}
}

func (t TailTest) FileInode(name string) (inode uint64) {
	f, err := os.OpenFile(t.path+"/"+name, os.O_RDONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal(err)
	}
	return st.Ino
}

func (t TailTest) StartTail(name string, config Config) *Tail {
	tail, err := TailFile(t.path+"/"+name, config)
	if err != nil {
		t.Fatal(err)
	}
	return tail
}

func (t TailTest) VerifyTailOutput(tail *Tail, lines []string, positions []int64, inodes []uint64) {
	if len(lines) != len(positions) {
		t.Fatal("len(lines) not equal len(positions)")
	}
	for idx, line := range lines {
		tailedLine, ok := <-tail.Lines
		if !ok {
			// tail.Lines is closed and empty.
			err := tail.Err()
			if err != nil {
				t.Fatalf("tail ended with error: %v", err)
			}
			t.Fatalf("tail ended early; expecting more: %v", lines[idx:])
		}
		if tailedLine == nil {
			t.Fatalf("tail.Lines returned nil; not possible")
		}
		// Note: not checking .Err as the `lines` argument is designed
		// to match error strings as well.
		if tailedLine.Text != line {
			t.Fatalf(
				"unexpected line/err from tail: "+
					"expecting <<%s>>>, but got <<<%s>>>",
				line, tailedLine.Text)
		}
		if tailedLine.Position != positions[idx] {
			t.Fatalf(
				"unexpected position/err from tail: "+
					"expecting <<%d>>>, but got <<<%d>>>",
				positions[idx], tailedLine.Position)
		}
		if idx < len(inodes) && tailedLine.Inode != inodes[idx] {
			t.Fatalf(
				"unexpected inode/err from tail: "+
					"expecting <<%d>>>, but got <<<%d>>>",
				inodes[idx], tailedLine.Inode)
		}
	}
	line, ok := <-tail.Lines
	if ok {
		t.Fatalf("more content from tail: %+v", line)
	}
}
