package src

import (
	"log"
	"reflect"

	"github.com/hamba/avro"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
)

type Model interface {
	Key() string
	Table() string
}

type ModelInfo struct {
	Table  string
	Schema avro.Schema

	// FieldMap translates from the golang field name to the corresponding
	// column name in SingleStore
	FieldMap map[string]string
}

var Models []ModelInfo

func init() {
	models := []Model{
		&AccessKey{},
		&AccountChange{},
		&Account{},
		&ActionReceiptAction{},
		&ActionReceiptInputData{},
		&ActionReceiptOutputData{},
		&ActionReceipt{},
		&Block{},
		&Chunk{},
		&DataReceipt{},
		&ExecutionOutcomeReceipt{},
		&ExecutionOutcome{},
		&Receipt{},
		&TransactionAction{},
		&Transaction{},
	}

	for _, model := range models {
		schema, fieldMap, err := GenerateSchemaAndFieldMap(model)
		if err != nil {
			panic(err)
		}
		Models = append(Models, ModelInfo{
			Table:    model.Table(),
			Schema:   schema,
			FieldMap: fieldMap,
		})
	}
}

func GenerateSchemaAndFieldMap(m interface{}) (avro.Schema, map[string]string, error) {
	mType := reflect.TypeOf(m)

	if mType.Kind() == reflect.Ptr {
		mType = mType.Elem()
	}

	if mType.Kind() != reflect.Struct {
		return nil, nil, errors.New("can only generate Avro schema for a struct")
	}

	fields := make([]*avro.Field, 0, mType.NumField())
	fieldMap := make(map[string]string)
	for i := 0; i < mType.NumField(); i++ {
		f := mType.Field(i)
		fType := f.Type

		fieldMap[f.Name] = strcase.ToSnake(f.Name)

		var schemaType avro.Type
		var nullable bool

		if fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
			nullable = true
		}

		switch fType.Kind() {
		case reflect.String:
			schemaType = avro.String
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			schemaType = avro.Int
		case reflect.Int64:
			schemaType = avro.Long
		case reflect.Float32:
			schemaType = avro.Float
		case reflect.Float64:
			schemaType = avro.Double
		case reflect.Bool:
			schemaType = avro.Boolean
		default:
			log.Fatalf("type not supported: %s", fType.Kind())
		}

		var fieldSchema avro.Schema = avro.NewPrimitiveSchema(schemaType, nil)
		var err error
		if nullable {
			fieldSchema, err = avro.NewUnionSchema([]avro.Schema{fieldSchema, &avro.NullSchema{}})
			if err != nil {
				return nil, nil, err
			}
		}

		field, err := avro.NewField(f.Name, fieldSchema, avro.NoDefault)
		if err != nil {
			return nil, nil, err
		}
		fields = append(fields, field)
	}

	schema, err := avro.NewRecordSchema(mType.Name(), "com.singlestore", fields)
	if err != nil {
		return nil, nil, err
	}

	return schema, fieldMap, err
}

type AccessKey struct {
	PublicKey             string
	AccountId             string
	CreatedByReceiptId    *string
	DeletedByReceiptId    *string
	PermissionKind        string
	LastUpdateBlockHeight string
}

func (m *AccessKey) Key() string {
	return m.PublicKey + ":" + m.AccountId
}

func (m *AccessKey) Table() string {
	return "access_keys"
}

type AccountChange struct {
	Id                              string
	AffectedAccountId               string
	ChangedInBlockTimestamp         string
	ChangedInBlockHash              string
	CausedByTransactionHash         *string
	CausedByReceiptId               *string
	UpdateReason                    string
	AffectedAccountNonstakedBalance string
	AffectedAccountStakedBalance    string
	AffectedAccountStorageUsage     string
}

func (m *AccountChange) Key() string {
	return m.Id
}

func (m *AccountChange) Table() string {
	return "account_changes"
}

type Account struct {
	Id                    string
	AccountId             string
	CreatedByReceiptId    *string
	DeletedByReceiptId    *string
	LastUpdateBlockHeight string
}

func (m *Account) Key() string {
	return m.Id
}

func (m *Account) Table() string {
	return "accounts"
}

type ActionReceiptAction struct {
	ReceiptId                       string
	IndexInActionReceipt            string
	ActionKind                      string
	Args                            string
	ReceiptPredecessorAccountId     string
	ReceiptReceiverAccountId        string
	ReceiptIncludedInBlockTimestamp string
}

func (m *ActionReceiptAction) Key() string {
	return m.ReceiptId + ":" + m.IndexInActionReceipt
}

func (m *ActionReceiptAction) Table() string {
	return "action_receipt_actions"
}

type ActionReceiptInputData struct {
	InputDataId      string
	InputToReceiptId string
}

func (m *ActionReceiptInputData) Key() string {
	return m.InputDataId + ":" + m.InputToReceiptId
}

func (m *ActionReceiptInputData) Table() string {
	return "action_receipt_input_data"
}

type ActionReceiptOutputData struct {
	OutputDataId        string
	OutputFromReceiptId string
	ReceiverAccountId   string
}

func (m *ActionReceiptOutputData) Key() string {
	return m.OutputDataId + ":" + m.OutputFromReceiptId
}

func (m *ActionReceiptOutputData) Table() string {
	return "action_receipt_output_data"
}

type ActionReceipt struct {
	ReceiptId       string
	SignerAccountId string
	SignerPublicKey string
	GasPrice        string
}

func (m *ActionReceipt) Key() string {
	return m.ReceiptId
}

func (m *ActionReceipt) Table() string {
	return "action_receipts"
}

type Block struct {
	BlockHeight     string
	BlockHash       string
	PrevBlockHash   string
	BlockTimestamp  string
	TotalSupply     string
	GasPrice        string
	AuthorAccountId string
}

func (m *Block) Key() string {
	return m.BlockHash
}

func (m *Block) Table() string {
	return "blocks"
}

type Chunk struct {
	IncludedInBlockHash string
	ChunkHash           string
	ShardId             string
	Signature           string
	GasLimit            string
	GasUsed             string
	AuthorAccountId     string
}

func (m *Chunk) Key() string {
	return m.ChunkHash
}

func (m *Chunk) Table() string {
	return "chunks"
}

type DataReceipt struct {
	DataId    string
	ReceiptId string
	Data      *string
}

func (m *DataReceipt) Key() string {
	return m.DataId
}

func (m *DataReceipt) Table() string {
	return "data_receipts"
}

type ExecutionOutcomeReceipt struct {
	ExecutedReceiptId       string
	IndexInExecutionOutcome string
	ProducedReceiptId       string
}

func (m *ExecutionOutcomeReceipt) Key() string {
	return m.ExecutedReceiptId + ":" + m.IndexInExecutionOutcome
}

func (m *ExecutionOutcomeReceipt) Table() string {
	return "execution_outcome_receipts"
}

type ExecutionOutcome struct {
	ReceiptId                string
	ExecutedInBlockHash      string
	ExecutedInBlockTimestamp string
	IndexInChunk             string
	GasBurnt                 string
	TokensBurnt              string
	ExecutorAccountId        string
	Status                   string
	ShardID                  string
}

func (m *ExecutionOutcome) Key() string {
	return m.ReceiptId
}

func (m *ExecutionOutcome) Table() string {
	return "execution_outcomes"
}

type Receipt struct {
	ReceiptId                     string
	IncludedInBlockHash           string
	IncludedInChunkHash           string
	IndexInChunk                  string
	IncludedInBlockTimestamp      string
	PredecessorAccountId          string
	ReceiverAccountId             string
	ReceiptKind                   string
	OriginatedFromTransactionHash string
}

func (m *Receipt) Key() string {
	return m.ReceiptId
}

func (m *Receipt) Table() string {
	return "receipts"
}

type TransactionAction struct {
	TransactionHash    string
	IndexInTransaction string
	ActionKind         string
	Args               string
}

func (m *TransactionAction) Key() string {
	return m.TransactionHash
}

func (m *TransactionAction) Table() string {
	return "transaction_actions"
}

type Transaction struct {
	TransactionHash              string
	IncludedInBlockHash          string
	IncludedInChunkHash          string
	IndexInChunk                 string
	BlockTimestamp               string
	SignerAccountId              string
	SignerPublicKey              string
	Nonce                        string
	ReceiverAccountId            string
	Signature                    string
	Status                       string
	ConvertedIntoReceiptId       string
	ReceiptConversionGasBurnt    string
	ReceiptConversionTokensBurnt string
}

func (m *Transaction) Key() string {
	return m.TransactionHash
}

func (m *Transaction) Table() string {
	return "transactions"
}
