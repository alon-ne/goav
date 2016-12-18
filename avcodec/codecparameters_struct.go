package avcodec

func (p *CodecParameters) CodecId() CodecId {
	return CodecId(p.codec_id)
}

func (p *CodecParameters) AvcodecParametersToContext() {

}