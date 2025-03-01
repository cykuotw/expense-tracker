// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package auth

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import "expense-tracker/frontend/views/layouts"

func Register() templ.Component {
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
		templ_7745c5c3_Var2 := templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
			templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
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
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"h-screen flex justify-center items-center\"><form class=\"flex flex-col justify-center items-center gap-3 py-5\" hx-post=\"/register\" hx-target-error=\"#feedback\" hx-indicator=\"#indicator\"><div class=\"text-2xl\">Welcome</div><label class=\"input input-bordered flex items-center gap-2 w-full\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 16 16\" fill=\"currentColor\" class=\"w-4 h-4 opacity-70\"><path d=\"M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM12.735 14c.618 0 1.093-.561.872-1.139a6.002 6.002 0 0 0-11.215 0c-.22.578.254 1.139.872 1.139h9.47Z\"></path></svg> <input required type=\"text\" id=\"firstname\" name=\"firstname\" class=\"grow\" placeholder=\"First Name\"></label> <label class=\"input input-bordered flex items-center gap-2 w-full\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 16 16\" fill=\"currentColor\" class=\"w-4 h-4 opacity-70\"><path d=\"M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM12.735 14c.618 0 1.093-.561.872-1.139a6.002 6.002 0 0 0-11.215 0c-.22.578.254 1.139.872 1.139h9.47Z\"></path></svg> <input required type=\"text\" id=\"lastname\" name=\"lastname\" class=\"grow\" placeholder=\"Last Name\"></label> <label class=\"input input-bordered flex items-center gap-2 w-full\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"16\" height=\"16\" fill=\"currentColor\" class=\"bi bi-suit-heart-fill\" viewBox=\"0 0 16 16\"><path d=\"M4 1c2.21 0 4 1.755 4 3.92C8 2.755 9.79 1 12 1s4 1.755 4 3.92c0 3.263-3.234 4.414-7.608 9.608a.513.513 0 0 1-.784 0C3.234 9.334 0 8.183 0 4.92 0 2.755 1.79 1 4 1\"></path></svg> <input type=\"text\" id=\"nickname\" name=\"nickname\" class=\"grow\" placeholder=\"Nickname (Optional)\"></label> <label class=\"input input-bordered flex items-center gap-2 w-full\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 16 16\" fill=\"currentColor\" class=\"w-4 h-4 opacity-70\"><path d=\"M2.5 3A1.5 1.5 0 0 0 1 4.5v.793c.026.009.051.02.076.032L7.674 8.51c.206.1.446.1.652 0l6.598-3.185A.755.755 0 0 1 15 5.293V4.5A1.5 1.5 0 0 0 13.5 3h-11Z\"></path><path d=\"M15 6.954 8.978 9.86a2.25 2.25 0 0 1-1.956 0L1 6.954V11.5A1.5 1.5 0 0 0 2.5 13h11a1.5 1.5 0 0 0 1.5-1.5V6.954Z\"></path></svg> <input hx-post=\"/register/validate/email\" hx-trigger=\"keyup changed delay:500ms\" hx-target=\"#email-msg\" hx-swap=\"outerHTML\" required type=\"text\" id=\"email\" name=\"email\" class=\"grow\" placeholder=\"example@youremail.com\"></label><div id=\"email-msg\" class=\"-my-2 text-xs w-full text-left text-green-500\"></div><label class=\"input input-bordered flex items-center gap-2 w-full\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 16 16\" fill=\"currentColor\" class=\"w-4 h-4 opacity-70\"><path fill-rule=\"evenodd\" d=\"M14 6a4 4 0 0 1-4.899 3.899l-1.955 1.955a.5.5 0 0 1-.353.146H5v1.5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1-.5-.5v-2.293a.5.5 0 0 1 .146-.353l3.955-3.955A4 4 0 1 1 14 6Zm-4-2a.75.75 0 0 0 0 1.5.5.5 0 0 1 .5.5.75.75 0 0 0 1.5 0 2 2 0 0 0-2-2Z\" clip-rule=\"evenodd\"></path></svg> <input hx-post=\"/register/validate/password\" hx-trigger=\"keyup changed delay:500ms\" hx-target=\"#password-msg\" hx-swap=\"outerHTML\" required type=\"password\" id=\"password\" name=\"password\" class=\"grow\" placeholder=\"Password\"></label><div id=\"password-msg\" class=\"-my-2 text-xs w-full text-left text-red-500\">* should contain number, uppercase, <br>lowercase, and special character</div><button type=\"submit\" class=\"btn btn-active btn-neutral btn-wide text-lg font-light\">Submit</button><div id=\"indicator\" class=\"htmx-indicator\"><div class=\"flex justify-center items-center w-full\"><span class=\"loading loading-spinner loading-md\"></span></div></div><div id=\"feedback\"></div></form></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			return templ_7745c5c3_Err
		})
		templ_7745c5c3_Err = layouts.Base().Render(templ.WithChildren(ctx, templ_7745c5c3_Var2), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
