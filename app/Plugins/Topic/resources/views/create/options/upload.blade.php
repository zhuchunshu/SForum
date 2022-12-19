
<div class="col-12">
    <div class="card">
        <div class="card-body">
            <h3 class="card-title">上传附件</h3>
            <div class="dropzone" id="dropzone-multiple">
                <div class="fallback">
                    <input name="file" type="file"  multiple  />
                </div>
                <div class="dz-message">
                    <h3 class="dropzone-msg-title">选择文件</h3>
                    <span class="dropzone-msg-desc">支持选择多个</span>
                    <div class="dz-size" data-dz-size></div>
                </div>
            </div>
        </div>
    </div>
    <script src="{{file_hash('tabler/libs/dropzone/dist/dropzone-min.js')}}" defer></script>
    <script defer>
        // @formatter:off
        document.addEventListener("DOMContentLoaded", function() {
            new Dropzone("#dropzone-multiple",{
                url:"/topic/create/upload"
            })
        })
    </script>
</div>