/*

   account.go
       Account (Signed) Endpoints for Binance Exchange API

*/
package binance

import (
	"fmt"
	"strconv"
	"time"
)

// Get Basic Account Information
func (b *Binance) GetAccountInfo() (account Account, err error) {

	reqUrl := fmt.Sprintf("api/v3/account")

	_, err = b.client.do("GET", reqUrl, "", true, &account)
	if err != nil {
		return
	}

	return
}

// Filter Basic Account Information To Retrieve Current Holdings
func (b *Binance) GetPositions() (positions []Balance, err error) {

	reqUrl := fmt.Sprintf("api/v3/account")
	account := Account{}

	_, err = b.client.do("GET", reqUrl, "", true, &account)
	if err != nil {
		return
	}

	positions = make([]Balance, len(account.Balances))
	i := 0

	for _, balance := range account.Balances {
		if balance.Free != 0.0 || balance.Locked != 0.0 {
			positions[i] = balance
			i++
		}
	}

	return positions[:i], nil
}

// Place a Limit Order
func (b *Binance) PlaceLimitOrder(l LimitOrder) (res PlacedOrder, err error) {

	err = l.ValidateLimitOrder()
	if err != nil {
		return
	}

	reqUrl := fmt.Sprintf("api/v3/order?symbol=%s&side=%s&type=%s&timeInForce=%s&quantity=%f&price=%.8f&recvWindow=%d", l.Symbol, l.Side, l.Type, l.TimeInForce, l.Quantity, l.Price, l.RecvWindow)

	_, err = b.client.do("POST", reqUrl, "", true, &res)
	if err != nil {
		return
	}

	return
}

// Place a TEST Limit Order
func (b *Binance) PlaceTestLimitOrder(l LimitOrder) (res PlacedOrder, err error) {

	err = l.ValidateLimitOrder()
	if err != nil {
		return
	}

	reqUrl := fmt.Sprintf("api/v3/order/test?symbol=%s&side=%s&type=%s&timeInForce=%s&quantity=%f&price=%.8f&recvWindow=%d", l.Symbol, l.Side, l.Type, l.TimeInForce, l.Quantity, l.Price, l.RecvWindow)

	_, err = b.client.do("POST", reqUrl, "", true, &res)
	if err != nil {
		return
	}

	// Query until the order is fulfilled
	maxTries := 300
	i := 0

	var query OrderQuery
	query.OrderId = res.OrderId
	query.Symbol = res.Symbol

	for {

		// Delete the order
		if i == maxTries {

			_, err3 := b.CancelOrder(query)

			if err3 != nil {
				return res, err3
			}

			// TODO: Call to delete the order on Binance
			return res, fmt.Errorf("Timeout fulfilling %s (%d attempts)", res.Symbol, i)
		}

		order, err2 := b.CheckOrder(query)

		// Return if an error is received from querying the orderID
		if err2 != nil {
			return res, err2
		}

		// If the order is marked as complete, return
		if order.Status == "FILLED" {
			res.Fills = make([]PlacedOrderFills, 1)
			res.Fills[0].Price = strconv.FormatFloat(order.Price, 'f', 10, 64)
			res.Fills[0].Qty = strconv.FormatFloat(order.ExecutedQty, 'f', 10, 64)

			return res, err
		}

		// Sleep, query again, until maxTries hit
		time.Sleep(time.Second * 1)
		i++

	}

	/*
		res.ClientOrderId = "test" + strconv.FormatInt(time.Now().Unix(), 10)
		res.Symbol = m.Symbol
		res.TransactTime = time.Now().Unix()
		res.OrderId = time.Now().Unix()

		// Return the "expected" data for the test transaction
		res.Fills = make([]PlacedOrderFills, 1)

		// Market price, generally differs from the quoted ticker, reflect to simulate a "real" transaction
		res.Fills[0].Price = strconv.FormatFloat(m.Price*1.025, 'f', 10, 64)
		res.Fills[0].Qty = strconv.FormatFloat(m.Quantity, 'f', 10, 64)
		res.Fills[0].Commission = strconv.FormatFloat(m.Price*0.005, 'f', 10, 64)
		res.Fills[0].CommissionAsset = "BNB"
		res.Fills[0].TradeId = time.Now().Unix()
	*/

	return
}

// Place a Market Order
func (b *Binance) PlaceMarketOrder(m MarketOrder) (res PlacedOrder, err error) {

	err = m.ValidateMarketOrder()
	if err != nil {
		return
	}

	// Return the FULL order-response by default, required to retrieve the price bought at.
	reqUrl := fmt.Sprintf("api/v3/order?symbol=%s&side=%s&type=%s&quantity=%f&recvWindow=%d&newOrderRespType=FULL", m.Symbol, m.Side, m.Type, m.Quantity, m.RecvWindow)

	_, err = b.client.do("POST", reqUrl, "", true, &res)
	if err != nil {
		return
	}

	return
}

// Place a TEST Market Order
func (b *Binance) PlaceTestMarketOrder(m TestMarketOrder) (res PlacedOrder, err error) {

	err = m.ValidateMarketOrder()
	if err != nil {
		return
	}

	// Return the FULL order-response by default, required to retrieve the price bought at.
	reqUrl := fmt.Sprintf("api/v3/order/test?symbol=%s&side=%s&type=%s&quantity=%f&recvWindow=%d&newOrderRespType=FULL", m.Symbol, m.Side, m.Type, m.Quantity, m.RecvWindow)

	_, err = b.client.do("POST", reqUrl, "", true, &res)
	if err != nil {
		return
	}

	// Return "TEST" data to simulate a transaction
	/*
		{"symbol":"NANOBTC","orderId":21420869,"clientOrderId":"QmntW66WkDLDw6yUUuqPHb","price":"0.00000000","origQty":"2.00000000","executedQty":"2.00000000","status":"FILLED","timeInForce":"GTC","type":"MARKET","side":"BUY","stopPrice":"0.00000000","icebergQty":"0.00000000","time":1526886603112,"isWorking":true}
	*/

	res.ClientOrderId = "test" + strconv.FormatInt(time.Now().Unix(), 10)
	res.Symbol = m.Symbol
	res.TransactTime = time.Now().Unix()
	res.OrderId = time.Now().Unix()

	// Return the "expected" data for the test transaction
	res.Fills = make([]PlacedOrderFills, 1)

	// Market price, generally differs from the quoted ticker, reflect to simulate a "real" transaction
	res.Fills[0].Price = strconv.FormatFloat(m.Price*1.025, 'f', 10, 64)
	res.Fills[0].Qty = strconv.FormatFloat(m.Quantity, 'f', 10, 64)
	res.Fills[0].Commission = strconv.FormatFloat(m.Price*0.005, 'f', 10, 64)
	res.Fills[0].CommissionAsset = "BNB"
	res.Fills[0].TradeId = time.Now().Unix()

	return
}

// Cancel an Order
func (b *Binance) CancelOrder(query OrderQuery) (order CanceledOrder, err error) {

	err = query.ValidateOrderQuery()
	if err != nil {
		return
	}

	reqUrl := fmt.Sprintf("api/v3/order?symbol=%s&orderId=%d&recvWindow=%d", query.Symbol, query.OrderId, query.RecvWindow)

	_, err = b.client.do("DELETE", reqUrl, "", true, &order)
	if err != nil {
		return
	}

	return
}

// Check the Status of an Order
func (b *Binance) CheckOrder(query OrderQuery) (status OrderStatus, err error) {

	err = query.ValidateOrderQuery()
	if err != nil {
		return
	}

	reqUrl := fmt.Sprintf("api/v3/order?symbol=%s&orderId=%d&origClientOrderId=%s&recvWindow=%d", query.Symbol, query.OrderId, query.RecvWindow)

	_, err = b.client.do("GET", reqUrl, "", true, &status)
	if err != nil {
		return
	}

	return
}

// Retrieve All Open Orders
func (b *Binance) GetAllOpenOrders() (orders []OrderStatus, err error) {
	_, err = b.client.do("GET", "api/v3/openOrders", "", true, &orders)

	if err != nil {
		return
	}

	return
}

// Retrieve All Open Orders for a given symbol
func (b *Binance) GetOpenOrders(query OpenOrdersQuery) (orders []OrderStatus, err error) {

	err = query.ValidateOpenOrdersQuery()
	if err != nil {
		return
	}
	reqUrl := fmt.Sprintf("api/v3/openOrders?symbol=%s&recvWindow=%d", query.Symbol, query.RecvWindow)
	_, err = b.client.do("GET", reqUrl, "", true, &orders)
	if err != nil {
		return
	}

	return
}

// Get all account orders; active, canceled, or filled.
func (b *Binance) GetAllOrders(query AllOrdersQuery) (orders []OrderStatus, err error) {
	err = query.ValidateAllOrdersQuery()
	if err != nil {
		return
	}
	reqUrl := fmt.Sprintf("api/v3/allOrders?symbol=%s&recvWindow=%d&limit=%d", query.Symbol, query.RecvWindow, query.Limit)
	if query.OrderId != 0 {
		reqUrl += fmt.Sprintf("&orderId=%d", query.OrderId)
	}
	_, err = b.client.do("GET", reqUrl, "", true, &orders)
	if err != nil {
		return
	}

	return
}

// Retrieves all trades
func (b *Binance) GetTrades(symbol string) (trades []Trade, err error) {

	reqUrl := fmt.Sprintf("api/v3/myTrades?symbol=%s", symbol)

	_, err = b.client.do("GET", reqUrl, "", true, &trades)

	if err != nil {
		return
	}
	return
}

func (b *Binance) GetTradesFromOrder(symbol string, id int64) (matchingTrades []Trade, err error) {

	reqUrl := fmt.Sprintf("api/v3/myTrades?symbol=%s", symbol)

	var trades []Trade
	_, err = b.client.do("GET", reqUrl, "", true, &trades)
	if err != nil {
		return
	}

	for _, t := range trades {
		if t.OrderId == id {
			matchingTrades = append(matchingTrades, t)
		}
	}
	return
}

//
// Retrieves all withdrawals
func (b *Binance) GetWithdrawHistory() (withdraws WithdrawList, err error) {

	reqUrl := fmt.Sprintf("wapi/v3/withdrawHistory.html")

	_, err = b.client.do("GET", reqUrl, "", true, &withdraws)
	if err != nil {
		return
	}
	return
}

//
// Retrieves all deposits
func (b *Binance) GetDepositHistory() (deposits DepositList, err error) {

	reqUrl := fmt.Sprintf("wapi/v3/depositHistory.html")

	_, err = b.client.do("GET", reqUrl, "", true, &deposits)
	if err != nil {
		return
	}
	return
}
