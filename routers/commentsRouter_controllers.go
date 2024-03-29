package routers
  
import (
        "github.com/astaxie/beego"
        "github.com/astaxie/beego/context/param"
)

func init() {

        beego.GlobalControllerRouter["devday-order/controllers:OrderController"] = append(beego.GlobalControllerRouter["devday-order/controllers:OrderController"],
                beego.ControllerComments{
                        Method:           "Post",
                        Router:           `/`,
                        AllowHTTPMethods: []string{"post"},
                        MethodParams:     param.Make(),
                        Params:           nil})

        beego.GlobalControllerRouter["devday-order/controllers:OrderController"] = append(beego.GlobalControllerRouter["devday-order/controllers:OrderController"],
                beego.ControllerComments{
                        Method:           "Get",
                        Router:           `/`,
                        AllowHTTPMethods: []string{"get"},
                        MethodParams:     param.Make(),
                        Params:           nil})
}
