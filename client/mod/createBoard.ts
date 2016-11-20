import AccountFormView, { newRequest } from './common'
import { inputValue } from '../util'

// Panel view for creating boards
export default class BoardCreationPanel extends AccountFormView {
	constructor() {
		super({ tag: "form" }, () =>
			this.onSubmit())
		this.renderPublicForm("/forms/createBoard")
	}

	private async onSubmit() {
		const req = newRequest()
		req["name"] = inputValue(this.el, 'boardName')
		req["title"] = inputValue(this.el, 'boardTitle')
		this.injectCaptcha(req)

		this.postResponse("/admin/createBoard", req)
	}
}
