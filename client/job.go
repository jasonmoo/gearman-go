// Copyright 2011 Xing Xing <mikespook@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package client

import (
    "bytes"
    "github.com/jasonmoo/gearman-go/common"
)

const (
    // Job type
    // JOB_NORMAL | JOB_BG means a normal level job run in background
    // normal level
    JOB_NORMAL = 0
    // background job
    JOB_BG = 1
    // low level
    JOB_LOW = 2
    // high level
    JOB_HIGH = 4
)

// An error handler
type ErrorHandler func(error)

// Client side job
type Job struct {
    Data                []byte
    Handle, UniqueId    string
    magicCode, DataType uint32
}

// Create a new job
func newJob(magiccode, datatype uint32, data []byte) (job *Job) {
    return &Job{magicCode: magiccode,
    DataType: datatype,
    Data:     data}
}

// Decode a job from byte slice
func decodeJob(data []byte) (job *Job, err error) {
    if len(data) < 12 {
        return nil, common.Errorf("Invalid data: %V", data)
    }
    datatype := common.BytesToUint32([4]byte{data[4], data[5], data[6], data[7]})
    l := common.BytesToUint32([4]byte{data[8], data[9], data[10], data[11]})
    if len(data[12:]) != int(l) {
        return nil, common.Errorf("Invalid data: %V", data)
    }
    data = data[12:]
    var handle string
    switch datatype {
        case common.WORK_DATA, common.WORK_WARNING, common.WORK_STATUS,
        common.WORK_COMPLETE, common.WORK_FAIL, common.WORK_EXCEPTION:
        i := bytes.IndexByte(data, '\x00')
        if i != -1 {
            handle = string(data[:i])
            data = data[i + 1:]
        }
    }

    return &Job{magicCode: common.RES,
        DataType: datatype,
        Data:     data,
        Handle:   handle}, nil
}

// Encode a job to byte slice
func (job *Job) Encode() (data []byte) {
    l := len(job.Data)
    tl := l + 12
    data = make([]byte, tl)

    magiccode := common.Uint32ToBytes(job.magicCode)
    datatype := common.Uint32ToBytes(job.DataType)
    datalength := common.Uint32ToBytes(uint32(l))

    for i := 0; i < tl; i ++ {
        switch {
        case i < 4:
            data[i] = magiccode[i]
        case i < 8:
            data[i] = datatype[i - 4]
        case i < 12:
            data[i] = datalength[i - 8]
        default:
            data[i] = job.Data[i - 12]
        }
    }
    // Alternative
    /*
    data = append(data, magiccode[:] ...)
    data = append(data, datatype[:] ...)
    data = append(data, datalength[:] ...)
    data = append(data, job.Data ...)
    */
    return
}

// Extract the job's result.
func (job *Job) Result() (data []byte, err error) {
    switch job.DataType {
    case common.WORK_FAIL:
        job.Handle = string(job.Data)
        return nil, common.ErrWorkFail
    case common.WORK_EXCEPTION:
        err = common.ErrWorkException
        fallthrough
    case common.WORK_COMPLETE:
        s := bytes.SplitN(job.Data, []byte{'\x00'}, 2)
        if len(s) != 2 {
            return nil, common.Errorf("Invalid data: %V", job.Data)
        }
        job.Handle = string(s[0])
        data = s[1]
    default:
        err = common.ErrDataType
    }
    return
}

// Extract the job's update
func (job *Job) Update() (data []byte, err error) {
    if job.DataType != common.WORK_DATA && job.DataType != common.WORK_WARNING {
        err = common.ErrDataType
        return
    }
    s := bytes.SplitN(job.Data, []byte{'\x00'}, 2)
    if len(s) != 2 {
        err = common.ErrInvalidData
        return
    }
    if job.DataType == common.WORK_WARNING {
        err = common.ErrWorkWarning
    }
    job.Handle = string(s[0])
    data = s[1]
    return
}
