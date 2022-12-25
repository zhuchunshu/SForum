<div class="modal modal-blur fade" id="modal-report" tabindex="-1" role="dialog" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered" role="document">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title" id="modal-report-title"></h5>
                <button type="button" modal-click="close" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
                <div class="row mb-3 align-items-end">
                    <div class="col-auto">
                        <label class="form-label">举报原因</label>
                        <select class="form-select" id="modal-report-select">
                            <option value="水贴">水贴</option>
                            <option value="广告">广告</option>
                            <option value="引战">引战</option>
                            <option value="违法">违法</option>
                            <option value="翻墙">翻墙</option>
                            <option value="政治">政治</option>
                            <option value="其他">其他</option>
                        </select>
                    </div>
                    <div class="col">
                        <label class="form-label">标题</label>
                        <input type="text" id="modal-report-input-title" class="form-control" />
                        <input type="hidden" value="" id="modal-report-input-type" />
                        <input type="hidden" value="" id="modal-report-input-type-id" />
                        <input type="hidden" value="" id="modal-report-input-url" />
                    </div>
                </div>
                <div>
                    <label class="form-label">详细说明</label>
                    <textarea class="form-control" id="modal-report-input-content"></textarea>
                    <small><b style="color: red">支持markdown</b></small>
                </div>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn me-auto" data-bs-dismiss="modal" modal-click="close">Close</button>
                <button type="button" class="btn btn-primary" data-bs-dismiss="modal" id="modal-report-submit">提交</button>
            </div>
        </div>
    </div>
</div>