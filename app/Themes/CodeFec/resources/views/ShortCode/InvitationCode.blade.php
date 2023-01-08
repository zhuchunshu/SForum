<div class="row justify-content-center">
    <div class="col-10">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">邀请码</h3>
            </div>

            <div class="card-table table-responsive">
                <table class="table table-vcenter">
                    <thead>
                    <tr>
                        <th>邀请码</th>
                        <th>使用者</th>
                    </tr>
                    </thead>
                    <tbody>
                    @if(!count($codes))
                        <tr>
                            <td class="text-muted">无</td>
                            <td class="text-muted">无</td>
                        </tr>
                    @else
                        @foreach($codes as $data)
                            <tr>
                                <td>
                                    {{$data->code}}
                                    <a  class="ms-1" core-click="copy" copy-content="{{$data->code}}" message="短代码复制成功!">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/link -->
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-copy" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <rect x="8" y="8" width="12" height="12" rx="2"></rect>
                                            <path d="M16 8v-2a2 2 0 0 0 -2 -2h-8a2 2 0 0 0 -2 2v8a2 2 0 0 0 2 2h2"></path>
                                        </svg>
                                    </a>
                                </td>
                                <td>
                                    @if($data->status)
                                        <a href="/users/{{$data->user->id}}.html">
                                            <span class="avatar avatar-rounded" style="background-image: url({{super_avatar($data->user)}})"></span>
                                        </a>
                                    @else
                                        未使用
                                    @endif
                                </td>
                            </tr>
                        @endforeach
                    @endif
                    </tbody>
                </table>
            </div>

        </div>
    </div>
</div>