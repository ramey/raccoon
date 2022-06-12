package serialization

// type Serializer interface {
// 	Serialize(m interface{}) ([]byte, error)
// }

type SerializeFunc func(m interface{}) ([]byte, error)
