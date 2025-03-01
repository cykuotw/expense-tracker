// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func NewGroup() templ.Component {
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
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"flex justify-center items-center py-5 h-screen md:h-auto\"><form class=\"flex flex-col justify-center items-center max-w-md\" hx-post=\"/create_group\" hx-target-error=\"#feedback\" hx-indicator=\"#indicator\"><div class=\"flex flex-col py-5 text-3xl\">Create New Group</div><div class=\"w-full\"><div class=\"label\"><label class=\"label-text\">Group Name</label></div><input type=\"text\" id=\"groupname\" name=\"groupname\" required class=\"grow input input-bordered flex items-center w-full\" placeholder=\"ie: My Group\"></div><div class=\"w-full\"><div class=\"label\"><label class=\"label-text\">Group Description</label></div><input type=\"text\" id=\"description\" name=\"description\" class=\"grow input input-bordered flex items-center w-full\" placeholder=\"Description (optional)\"></div><div class=\"w-full\"><div class=\"label\"><label class=\"label-text\">Main Currency</label></div><select class=\"select select-bordered w-full\" name=\"currency\" id=\"currency\"><option value=\"CAD\">CAD</option> <option value=\"USD\">USD</option> <option value=\"NTD\">NTD</option></select></div><div class=\"w-full py-5\"><button type=\"submit\" class=\"btn btn-active btn-neutral btn-wide text-lg font-light\">Create Group</button></div><div id=\"indicator\" class=\"htmx-indicator\"><div class=\"flex justify-center items-center w-full\"><span class=\"loading loading-spinner loading-md\"></span></div></div><div id=\"feedback\"></div></form></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
