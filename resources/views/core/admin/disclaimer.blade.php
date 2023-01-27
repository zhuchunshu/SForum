@if(!cache()->has('admin.core.disclaimer'))
    <div x-data="hookmodal" class="modal modal-blur fade" id="admin-ui-hook-core-disclaimer" tabindex="-1" role="dialog">
        <div class="modal-dialog  modal-xl modal-dialog-centered modal-dialog-scrollable" role="document">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">SForum免责声明</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                </div>
                <div class="modal-body">
                    @include("core.install.disclaimer")
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn me-auto" data-bs-dismiss="modal">Close</button>
                    <button type="button" class="btn btn-primary" x-on:click="await agree()" data-bs-dismiss="modal">同意</button>
                </div>
            </div>
        </div>
    </div>
    <script>
       $(function(){
           $('#admin-ui-hook-core-disclaimer').modal('show')
       })
    </script>
    <script>
        document.addEventListener('alpine:init', () => {
            Alpine.data('hookmodal', () => ({
                async agree() {
                    let response = await fetch('/api/admin/agree.disclaimer?_token='+csrf_token)

                    return await response.text()
                }
            }))
        })
    </script>
@endif