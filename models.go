package main

type Model interface {
	Key() string
	Record() []string
}

func fmtStringPtr(s *string) string {
	if s == nil {
		return "\\N"
	}
	return *s
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

func (m *AccessKey) Record() []string {
	return []string{
		m.PublicKey,
		m.AccountId,
		fmtStringPtr(m.CreatedByReceiptId),
		fmtStringPtr(m.DeletedByReceiptId),
		m.PermissionKind,
		m.LastUpdateBlockHeight,
	}
}

type AccountChange struct {
	Id                              string
	AffectedAccountId               string
	ChangedInBlockTimestamp         string
	ChangedInBlockHash              string
	CausedByTransactionHash         string
	CausedByReceiptId               string
	UpdateReason                    string
	AffectedAccountNonstakedBalance string
	AffectedAccountStakedBalance    string
	AffectedAccountStorageUsage     string
}

func (m *AccountChange) Key() string {
	return m.Id
}

func (m *AccountChange) Record() []string {
	return []string{
		m.Id,
		m.AffectedAccountId,
		m.ChangedInBlockTimestamp,
		m.ChangedInBlockHash,
		m.CausedByTransactionHash,
		m.CausedByReceiptId,
		m.UpdateReason,
		m.AffectedAccountNonstakedBalance,
		m.AffectedAccountStakedBalance,
		m.AffectedAccountStorageUsage,
	}
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

func (m *Account) Record() []string {
	return []string{
		m.Id,
		m.AccountId,
		fmtStringPtr(m.CreatedByReceiptId),
		fmtStringPtr(m.DeletedByReceiptId),
		m.LastUpdateBlockHeight,
	}
}

type ActionReceiptAction struct {
	ReceiptId            string
	IndexInActionReceipt string
	ActionKind           string
	Args                 string
}

func (m *ActionReceiptAction) Key() string {
	return m.ReceiptId + ":" + m.IndexInActionReceipt
}

func (m *ActionReceiptAction) Record() []string {
	return []string{
		m.ReceiptId,
		m.IndexInActionReceipt,
		m.ActionKind,
		m.Args,
	}
}

type ActionReceiptInputData struct {
	InputDataId      string
	InputToReceiptId string
}

func (m *ActionReceiptInputData) Key() string {
	return m.InputDataId + ":" + m.InputToReceiptId
}

func (m *ActionReceiptInputData) Record() []string {
	return []string{
		m.InputDataId,
		m.InputToReceiptId,
	}
}

type ActionReceiptOutputData struct {
	OutputDataId        string
	OutputFromReceiptId string
	ReceiverAccountId   string
}

func (m *ActionReceiptOutputData) Key() string {
	return m.OutputDataId + ":" + m.OutputFromReceiptId
}

func (m *ActionReceiptOutputData) Record() []string {
	return []string{
		m.OutputDataId,
		m.OutputFromReceiptId,
		m.ReceiverAccountId,
	}
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

func (m *ActionReceipt) Record() []string {
	return []string{
		m.ReceiptId,
		m.SignerAccountId,
		m.SignerPublicKey,
		m.GasPrice,
	}
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

func (m *Block) Record() []string {
	return []string{
		m.BlockHeight,
		m.BlockHash,
		m.PrevBlockHash,
		m.BlockTimestamp,
		m.TotalSupply,
		m.GasPrice,
		m.AuthorAccountId,
	}
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

func (m *Chunk) Record() []string {
	return []string{
		m.IncludedInBlockHash,
		m.ChunkHash,
		m.ShardId,
		m.Signature,
		m.GasLimit,
		m.GasUsed,
		m.AuthorAccountId,
	}
}

type DataReceipt struct {
	DataId    string
	ReceiptId string
	Data      *string
}

func (m *DataReceipt) Key() string {
	return m.DataId
}

func (m *DataReceipt) Record() []string {
	return []string{
		m.DataId,
		m.ReceiptId,
		fmtStringPtr(m.Data),
	}
}

type ExecutionOutcomeReceipt struct {
	ExecutedReceiptId       string
	IndexInExecutionOutcome string
	ProducedReceiptId       string
}

func (m *ExecutionOutcomeReceipt) Key() string {
	return m.ExecutedReceiptId + ":" + m.IndexInExecutionOutcome
}

func (m *ExecutionOutcomeReceipt) Record() []string {
	return []string{
		m.ExecutedReceiptId,
		m.IndexInExecutionOutcome,
		m.ProducedReceiptId,
	}
}

type ExecutionOutcome struct {
	ReceiptId                string
	ExecutedInBlockHash      string
	ExecutedInBlockTimestamp string
	ExecutedInChunkHash      string
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

func (m *ExecutionOutcome) Record() []string {
	return []string{
		m.ReceiptId,
		m.ExecutedInBlockHash,
		m.ExecutedInBlockTimestamp,
		m.ExecutedInChunkHash,
		m.IndexInChunk,
		m.GasBurnt,
		m.TokensBurnt,
		m.ExecutorAccountId,
		m.Status,
		m.ShardID,
	}
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

func (m *Receipt) Record() []string {
	return []string{
		m.ReceiptId,
		m.IncludedInBlockHash,
		m.IncludedInChunkHash,
		m.IndexInChunk,
		m.IncludedInBlockTimestamp,
		m.PredecessorAccountId,
		m.ReceiverAccountId,
		m.ReceiptKind,
		m.OriginatedFromTransactionHash,
	}
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

func (m *TransactionAction) Record() []string {
	return []string{
		m.TransactionHash,
		m.IndexInTransaction,
		m.ActionKind,
		m.Args,
	}
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

func (m *Transaction) Record() []string {
	return []string{
		m.TransactionHash,
		m.IncludedInBlockHash,
		m.IncludedInChunkHash,
		m.IndexInChunk,
		m.BlockTimestamp,
		m.SignerAccountId,
		m.SignerPublicKey,
		m.Nonce,
		m.ReceiverAccountId,
		m.Signature,
		m.Status,
		m.ConvertedIntoReceiptId,
		m.ReceiptConversionGasBurnt,
		m.ReceiptConversionTokensBurnt,
	}
}
