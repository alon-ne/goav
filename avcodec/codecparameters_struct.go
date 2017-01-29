package avcodec

func (p *CodecParameters) CodecId() CodecId {
	return CodecId(p.codec_id)
}

func (p* CodecParameters) CodecType() MediaType {
	return MediaType(p.codec_type)
}

func (p *CodecParameters) AvcodecParametersToContext() {

}