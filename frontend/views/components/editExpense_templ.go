// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"expense-tracker/types"
	"fmt"
	"github.com/google/uuid"
)

func EditExpense(expense types.ExpenseResponse) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"flex flex-row justify-center items-center py-5 w-screen\"><form class=\"flex flex-col justify-center items-center py-5 space-y-5 md:w-1/3 w-5/6 max-w-md\" hx-put=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(string(templ.URL(fmt.Sprintf("/update_expense?e=%s", expense.ID))))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 13, Col: 78}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" hx-target-error=\"#feedback\" hx-indicator=\"#indicator\"><div class=\"text-2xl\">Edit Expense</div><div class=\"flex flex-row justify-start items-start w-full\"><div class=\"w-full\" hx-get=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(string(templ.URL(fmt.Sprintf("/groupSelect/%s", expense.GroupId))))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 23, Col: 80}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" hx-trigger=\"load\" hx-swap=\"outerHTML\"><select class=\"select select-bordered w-full text-base text-center\"></select></div></div><div class=\"flex flex-row justify-start items-start w-full\"><div class=\"w-full\" hx-get=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var4 string
		templ_7745c5c3_Var4, templ_7745c5c3_Err = templ.JoinStringErrs(string(templ.URL(fmt.Sprintf("/expense_types/%s", expense.ExpenseType))))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 33, Col: 86}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" hx-trigger=\"load\" hx-swap=\"outerHTML\"><select class=\"select select-bordered w-3/5 text-base text-center\"></select></div></div><div class=\"flex flex-row justify-start items-start w-full\"><label class=\"input input-bordered flex items-center w-full\"><input type=\"text\" id=\"description\" name=\"description\" class=\"grow\" placeholder=\"Description\" value=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var5 string
		templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(expense.Description)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 48, Col: 33}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\"></label></div><div class=\"flex flex-row justify-start items-start w-full\"><select class=\"select select-bordered w-1/3 text-base text-center\" id=\"currency\" name=\"currency\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for _, currency := range []string{"CAD", "NTU", "USD"} {
			if currency == expense.Currency {
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<option selected>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var6 string
				templ_7745c5c3_Var6, templ_7745c5c3_Err = templ.JoinStringErrs(currency)
				if templ_7745c5c3_Err != nil {
					return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 60, Col: 34}
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var6))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</option>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
			} else {
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<option>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var7 string
				templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs(currency)
				if templ_7745c5c3_Err != nil {
					return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 62, Col: 25}
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</option>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</select> <label class=\"input input-bordered flex items-center w-full\"><input type=\"number\" id=\"total\" name=\"total\" class=\"grow\" step=\"0.001\" placeholder=\"0.00\" value=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var8 string
		templ_7745c5c3_Var8, templ_7745c5c3_Err = templ.JoinStringErrs(expense.Total.String())
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 74, Col: 36}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var8))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\"></label></div><div class=\"hidden\"><label style=\"display: inline-block;\" class=\"w-2/3 h-12 border border-gray-400 rounded-full bg-base-100 hover:bg-base-300\"><input type=\"file\" style=\"display: none;\"><div class=\"flex flex-row items-center justify-center h-full space-x-3\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\" class=\"w-6 h-6 opacity-70\" fill=\"currentColor\"><path d=\"M4,4H7L9,2H15L17,4H20A2,2 0 0,1 22,6V18A2,2 0 0,1 20,20H4A2,2 0 0,1 2,18V6A2,2 0 0,1 4,4M12,7A5,5 0 0,0 7,12A5,5 0 0,0 12,17A5,5 0 0,0 17,12A5,5 0 0,0 12,7M12,9A3,3 0 0,1 15,12A3,3 0 0,1 12,15A3,3 0 0,1 9,12A3,3 0 0,1 12,9Z\"></path></svg><p>Upload Receipt</p></div></label></div><div id=\"splitRule\" class=\"w-2/3\" hx-get=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var9 string
		templ_7745c5c3_Var9, templ_7745c5c3_Err = templ.JoinStringErrs(string(templ.URL(fmt.Sprintf("/split_rules?e=%s", expense.ID))))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `frontend/views/components/editExpense.templ`, Line: 95, Col: 76}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var9))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" hx-trigger=\"load\" hx-swap=\"outerHTML\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = SplitRule(types.GroupMember{
			UserID:   uuid.NewString(),
			Username: "user me",
		}, []types.GroupMember{
			{
				UserID:   uuid.NewString(),
				Username: "user 1",
			},
		}).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div><button id=\"submit\" type=\"submit\" class=\"btn btn-active btn-neutral btn-wide text-lg font-light\" disabled><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\" class=\"w-6 h-6 opacity-70\" fill=\"currentColor\"><path d=\"M9,20.42L2.79,14.21L5.62,11.38L9,14.77L18.88,4.88L21.71,7.71L9,20.42Z\"></path></svg> Update</button><div id=\"indicator\" class=\"htmx-indicator\"><div class=\"flex justify-center items-center w-full\"><span class=\"loading loading-spinner loading-md\"></span></div></div></form></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
