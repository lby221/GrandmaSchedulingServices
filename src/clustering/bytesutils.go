package clustering

func append_buffer(dst, src []byte, begindst, beginsrc, length int) {
	for index := begindst; index < begindst+length; index++ {
		dst[index] = src[index-begindst+beginsrc]
	}
}
