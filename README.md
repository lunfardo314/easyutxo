# EasyUTXO
A simple yet extendable, programmable and powerful UTXO ledger model with advanced concepts

EasyUTXO is a work in progress. This _readme_ is preliminary

Here is a [preliminary presentation](https://hackmd.io/@Evaldas/S14WHOKMi) of the **EasyFL** functional constraint language 

## Introduction

The _EasyUTXO_ is an implementation of the UTXO concept well-known in distributed ledger technologies, such as Bitcoin or 
IOTA (Stardust release). 

The _EasyUTXO_ model of the UTXO ledger intends to go beyond the Stardust ledger model, 
which is hardcoded and language-dependent to certain extent. The _EasyUTXO_ introduces **programmability** of the ledger transition 
function (transaction model), while same time preserving robustness and determinism of the UTXO ledger model 
by adding **formal verifiability** and **minimizing the global trust assumptions**. 
The reference code _EasyUTXO_ model is written in Go, however the transaction model (UTXO types/behavior) are completely 
platform independent.

The above is achieved by:
* building very generic and minimalistic transaction model (the minimization of assumptions)
* programming almost all the state transition logic as the _finite functional constraints_, expressed in the
[EasyFL](https://github.com/lunfardo314/easyfl) functional constraint language. 

The whole model of the state transition is finite and non-Turing complete, because EasyFL language does not allow loops and recursions.

This makes _EasyFL_ program (expression) and its specification is the same thing. With this trait of the *EasyUTXO* makes 
it possible **automatic prove** of the ledger properties, e.g. if it can achieve desired (or undesired) states.
For example the answer to the questions _Is it possible to spend funds without knowing private key?_ 
or _Is the output with specific constraint unlockable?_ can be automated. This is not possible with Turing-complete models
without adding formal specifications to the program code./

This way essentially all the Stardust UTXO types are expressed in provable _EasyFL_ constraints: 
all kind of transfers, chain constraints, aliases, NFTs, time lock, expiry conditions etc 
(by the time of writing, native tokens and foundries remains in future plans).

This also makes extension and programmability of the UTXO transactions by embedding new constraints right into the transaction (inline),
or extending library of constrains with new constraints, available globally.

The functional nature of the _EasyFL_ makes it easy to **extend the computation model** of the ledger even further, for example
with Turing-complete computations of _WebAssembly_, _EVM_, _Move_ or any other. 

The above makes _EasyUTXO_ a powerful yet safe architectural abstraction layer for any further developments of the UTXO ledgers.

## Transaction model

### Output
UTXO ledger consists of UTXOs (**U**nspent **T**ransa**X**ion **O**utputs). Each output is an array of _constraints_:
_(C1, C1m ..., CN)_. Each constraint _Ci_ is an _EasyFL_ expression (formula) in its compressed binary form.

For the output to be valid, all constraints of the output are evaluated. If all formulas does not panic and return _true_, 
the output is valid. If constraint evaluates to _false_ or panics, the whole output is invalidated.

Each constraint is evaluated by supplying to the *EasyFL* engine:
* global context of the transaction being validated
* local place (path) in the transaction where being evaluated constraint is located. This constraints the data in the local context. 
For example, constraint `addressED25519(0x12345...)` by knowing its location, reaches out the corresponding _unlock block_ or signature
to check if the signature is valid

Any data which appears in the output comes in the form of constraint, i.e. is runnable. Examples of constraints:
* `amount(1000)` check validity of the amount
* `timestamp(123456)` checks validity of the timestamp in the output
* `chainLock(0xaaaaaaaaaaaaaa)` check if output can be consumed, i.e. if the chain `0xaaaaaaaaaaaaa..` is transited in the same transaction

### Transaction
Ledger is updated in atomic units, called _transaction_. Each transaction consist of:
* _consumed outputs_, up to 255
* _produced outputs_, up to 255
* _transaction level data_, such as transaction timestamp, _input commitment_, user signature, provided constraint scripts etc

We distinguish full _validation context of the transaction_ and _transaction_ it self. 

_Validation context_ includes all consumed inputs in its entirety. It is necessary for validation of the transaction.
The _transaction_ itself only contains _inputIDs_ instead of consumed outputs. The _transaction_ is the data transferred over the wire
between nodes. To validate it, the node rebuild _validation context_ on its own copy of the ledger state.

### Validation of the transaction

The node runs the following steps fo each transaction received by the wire:
* rebuilds _validation context_
* it evaluates all constraints on each consumed output 
* it evaluates all constraints on each produced output 
* checks integrity of global constraints, which cannot be expressed in EasyFL (unbounded):
  * balances of `amount(N)` constraints must be equal on consumed and produced side
  * each outputs must have one of predefined locks (addresses) 
  * _checks validity of the input commitment_

The above loop is the only global assumption of the ledger. It is completely agnostic of what constraint are provided. 
This way UTXO behavior is encoded into the constraints on outputs. 

