// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package layouts

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func Base() templ.Component {
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
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!doctype html><html lang=\"en\" data-theme=\"cupcake\"><head><title>Expense Tracker</title><link rel=\"icon\" type=\"image/x-icon\" href=\"/public/favicon.ico\"><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><link rel=\"stylesheet\" href=\"/public/css/styles.css\"><script src=\"/public/js/htmx.min.js\"></script><script src=\"/public/js/query-3.7.1.min.js\"></script><link rel=\"stylesheet\" href=\"https://fonts.googleapis.com/css?family=Roboto\"></head><style>\n\t\t\tbody {\n\t\t\t\tfont-family:'Roboto', serif;\n\t\t\t}\n\t\t</style><body class=\"antialiased\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templ_7745c5c3_Var1.Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</body><script type=\"text/javascript\">\n\t\t\twindow.addEventListener('click', function(e) {\n\t\t\t\tdocument.querySelectorAll('.dropdown').forEach(function(dropdown) {\n\t\t\t\t\tif (!dropdown.contains(e.target)) {\n\t\t\t\t\t// Click was outside the dropdown, close it\n\t\t\t\t\tdropdown.open = false;\n\t\t\t\t\t}\n\t\t\t\t});\n\t\t\t});\n\t\t</script></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
