// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func splitEvent2() templ.Component {
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
		templ_7745c5c3_Err = splitEvent().Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<script type=\"text/javascript\">\n        const payerContainer = document.querySelector(\"#payer-container\")\n        const payerInput = document.querySelector(\"#payerSelector\")\n\n        const updatePayer = () => {\n            const payer = splitRuleSelector.value.split(\"-\")[0]\n            const payerOptions = [...payerInput.options].map(obj => obj.value)\n\n            if(payer == 'Unequally'){\n                payerContainer.classList.remove(\"hidden\")\n                payerInput.value = payerOptions[0]\n                return\n            }\n            \n            if(!payerContainer.classList.contains(\"hidden\"))\n                payerContainer.classList.add(\"hidden\")\n\n            if(payer === \"you\"){\n                payerInput.value = payerOptions[0]\n            }else if(payer === \"other\"){\n                payerInput.value = payerOptions[1]\n            }\n        }\n\n        splitRuleSelector.addEventListener(\"change\", ()=>{\n            updatePayer()\n        })\n    </script>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func splitEvent() templ.Component {
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
		templ_7745c5c3_Var2 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var2 == nil {
			templ_7745c5c3_Var2 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<script type=\"text/javascript\">\n\t\tconst splitRuleSelector = document.querySelector(\"#splitRuleSelector\")\n\t\tconst ledger = document.querySelector(\"#ledger\")\n\t\tconst ledgerShares = document.querySelectorAll(\"#ledger-share\")\n\t\tconst totalInput = document.querySelector(\"#total\")\n        const descInput = document.querySelector(\"#description\")\n\t\tconst indicator = document.querySelector(\"#split-indicator\")\n        const submitButton = document.querySelector(\"#submit\")\n\n        var ledgerShareDone = false\n\n\t\tconst updateLedger = () => {\n\t\t\tif(totalInput.value === '' || totalInput.value === totalInput.defaultValue)\n\t\t\t\treturn\n\t\t\tif(splitRuleSelector.value === \"Unequally\")\n\t\t\t\treturn\n\t\t\t\n\t\t\tconst total = totalInput.value\n\t\t\tconst peopleCount = ledgerShares.length\n\t\t\t\n\t\t\tif(total <= 0 || peopleCount <= 2)\n\t\t\t\treturn\n\t\t\t\n\t\t\tconst split = (total / peopleCount).toFixed(2)\n\t\t\tconst remaining = total - (split * (peopleCount - 1))\n\n\t\t\tconst randomIndex = Math.floor(Math.random() * peopleCount);\n\t\t\tledgerShares.forEach((share, index) => {\n\t\t\t\tif(index === randomIndex){\n\t\t\t\t\tshare.value = remaining\n\t\t\t\t}else{\n\t\t\t\t\tshare.value = split\n\t\t\t\t}\n\t\t\t})\n\t\t}\n\n\t\tconst updateSplitIndicator = () => {\t\t\t\n\t\t\tif(totalInput.value === '' || totalInput.value === totalInput.defaultValue){\n\t\t\t\tindicator.innerHTML = ''\n\t\t\t\tledgerShares.forEach((share) => {\n\t\t\t\t\tshare.value = ''\n\t\t\t\t})\n\t\t\t\treturn\n\t\t\t}\n\n\t\t\tconst currency = document.querySelector(\"#currency\").value\n\t\t\tconst total = Number(totalInput.value)\n\t\t\tlet sum = 0\n\t\t\tledgerShares.forEach((share) => {\n\t\t\t\tif(share.value !== '' || share.value !== share.defaultValue){\n\t\t\t\t\tsum += Number(share.value)\n\t\t\t\t}\n\t\t\t})\n\n\t\t\tif(sum === total){\n\t\t\t\tindicator.innerHTML = \"<p class=\\\"text-green-700\\\">Total $0 \" + currency + \" left.</p>\"\n                ledgerShareDone = true\n\t\t\t}else{\n\t\t\t\tindicator.innerHTML = \"<p class=\\\"text-red-700\\\">Total $\"+ (total - sum) +\" \" + currency + \" left.</p>\"\n                ledgerShareDone = false\n\t\t\t}\n            checkAllInput()\n\t\t}\n\n        const checkAllInput = () => {\n            let doneDesc = (descInput.value !== '' && descInput.value !== descInput.defaultValue)\n            let doneTotal = (totalInput.value !== '' && totalInput.value !== totalInput.defaultValue)\n            \n            let ledgerHidden = ledger.classList.contains(\"hidden\")\n            \n            let done = doneDesc && doneTotal\n            if(!ledgerHidden)\n                done = done && ledgerShareDone\n            \n            if(done)\n                submitButton.disabled = false\n            else\n                submitButton.disabled = true\n        }\n\n\t\tsplitRuleSelector.addEventListener(\"change\", (event) => {\n\t\t\tif(event.target.value === \"Unequally\"){\n\t\t\t\tledger.classList.remove(\"hidden\")\n\n\t\t\t\tledgerShares.forEach((share) => {\n\t\t\t\t\tshare.value = null\n\t\t\t\t})\n                ledgerShareDone = false\n\t\t\t}else{\n\t\t\t\tledger.classList.add(\"hidden\")\n\t\t\t\tupdateLedger()\n\t\t\t}\n            updateSplitIndicator()\n\t\t})\n\n        descInput.addEventListener(\"change\", () => {\n            checkAllInput()\n        })\n\n\t\ttotalInput.addEventListener(\"change\", () => {\n\t\t\tupdateLedger()\n\t\t\tupdateSplitIndicator()\n\t\t})\n\n\t\tledgerShares.forEach((share) => {\n\t\t\tshare.addEventListener(\"change\", updateSplitIndicator)\n\t\t})\n\n\t</script>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
