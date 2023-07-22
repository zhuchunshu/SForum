@extends('app')
@section('title','关键词管理')
@section('content')
    <div class="col-12">
        <div class="card" x-data="topickeywords">
            <div class="card-header">
                <h3 class="card-title">关键词管理</h3>
                <div class="card-actions">
                    <form action="" method="GET">
                        <div class="row g-2">
                            <div class="col">
                                <input name="q" type="text" class="form-control" placeholder="Search for…" />
                            </div>
                            <div class="col-auto">
                                <button type="submit" class="btn btn-icon" aria-label="Button">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                                        <circle cx="10" cy="10" r="7" />
                                        <line x1="21" y1="21" x2="15" y2="15" />
                                    </svg>
                                </button>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
            <div class="table-responsive">
                <table class="table table-vcenter card-table">
                    <thead>
                    <tr>
                        <th>#</th>
                        <th>名称</th>
                        <th>创建者</th>
                        <th class="w-2"></th>
                    </tr>
                    </thead>
                    <tbody>
                    @if($page->count())
                        @foreach($page as $data)
                            <tr>
                                <td>{{$data->id}}</td>
                                <td class="text-muted">
                                    {{$data->name}}
                                </td>
                                <td class="text-muted"><a href="/users/{{$data->user->id}}.html"
                                                          class="text-reset">{{$data->user->username}}</a></td>
                                <td>
                                    <button @@click="remove({{$data->id}})" class="btn btn-link">删除</button>
                                </td>
                            </tr>
                        @endforeach
                    @else
                        <tr class="text-center">
                            <td colspan="4">暂无</td>
                        </tr>
                    @endif
                    </tbody>
                </table>

            </div>
            @if($page->count())
                {!! make_page($page) !!}
            @endif
        </div>
    </div>
@endsection

@section('scripts')
    <script>
        document.addEventListener('alpine:init', () => {
            Alpine.data('topickeywords', () => ({

                remove(id) {
                    // 询问是否要删除
                    const confirmRemove = confirm('Are you sure you want to remove this entry?');
                    if (!confirmRemove) {
                        return;
                    }
                    // 发出fetch请求删除关键词
                    fetch("/admin/topic/keywords/" + id, {
                        method: 'DELETE',
                        headers: {
                            'Content-Type': 'application/json' // 设置适当的请求头
                        },
                        // 如果有需要的话，可以在body中传递数据，但大多数DELETE请求不需要body
                        body: JSON.stringify({_token: csrf_token}),
                    })
                        .then(response => {
                            if (!response.ok) {
                                swal('Bad Request', `Failed to delete keyword: ${response.status}`, 'error')
                                throw new Error('Network response was not ok');
                            }
                            return response.json(); // 如果服务器返回JSON响应，则解析为JSON
                        })
                        .then(data => {
                            // 处理成功的响应数据
                            swal("Good job!", `${data.result.msg}`, 'success')
                            setTimeout(() => {
                                window.location.reload();
                            }, 1200)
                        })
                        .catch(error => {
                            // 处理错误情况
                            console.error('删除失败', error);
                            swal("Good job!", `${data.result.msg}`, 'error')
                        });
                }
            }))
        })
    </script>
@endsection