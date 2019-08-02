package controllers

import (
	"devday-order/models"
	"encoding/json"
	"time"
	"fmt"
	"strconv"
	"github.com/astaxie/beego"
)

type OrderController struct {
	beego.Controller
}

func init() {

}

func (this *OrderController) Post() {

	var ob models.Order
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	orderID, err := models.AddOrderToMongoDB(ob)

	if err == nil {
		this.Data["json"] = map[string]string{"orderId": orderID}

		fmt.Printf("[%s] orderid: %s\n", time.Now().Format(time.UnixDate), orderID)
	} else {
		this.Data["json"] = map[string]string{"error": "order not added to MongoDB. Check logs: " + err.Error()}
		this.Ctx.Output.SetStatus(500)

		fmt.Printf("[%s] orderid: %s\n", time.Now().Format(time.UnixDate), orderID)
	}
	
	this.ServeJSON()
}

func (this *OrderController) Get() {

	var ob models.Order
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	orderCount, err := models.GetNumberOfOrdersInDB()

	if err == nil {
		this.Data["json"] = map[string]string{"orderCount": strconv.Itoa(orderCount), "timestamp": time.Now().String()}
	} else {
		this.Data["json"] = map[string]string{"error": "couldn't query order count. Check logs: " + err.Error()}
		this.Ctx.Output.SetStatus(500)
	}
	
	this.ServeJSON()
}
