/*
 * @Notice: edit notice here
 * @Author: zhulei
 * @Date: 2022-12-02 20:57:58
 * @LastEditors: zhulei
 * @LastEditTime: 2022-12-12 13:41:18
 */
package pool

import "errors"

var (
	//ErrClosed 连接池已经关闭Error
	ErrClosed = errors.New("pool is closed")
)

// Pool 基本方法
type Pool interface {
	Get() (interface{}, error)

	Put(interface{}) error

	Close(interface{}) error

	Release()

	Len() int
}
