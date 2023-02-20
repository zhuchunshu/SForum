@extends('app')
@section('title','版主列表')
@section('content')
    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">版主列表</h3>
                <div class="card-actions">
                    <a href="./moderator/create">新增版主</a>
                </div>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover card-table">
                        <thead>
                        <tr>
                            <th>姓名</th>
                            <th>绑定标签(论坛)</th>
                            <th>绑定用户</th>
                            <th>操作</th>
                        </tr>
                        </thead>
                        <tbody x-data="moderator">
                        @foreach($page as $moderator)
                            <tr>
                                <td>{{$moderator->id}}</td>
                                <td>
                                    <span class="badge bg-primary-lt">{!! $moderator->tag->icon !!}</span> {{$moderator->tag->name}}
                                </td>
                                <td>
                                    <div><a href="/users/{{$moderator->user_id}}.html"
                                            class="avatar avatar-sm avatar-rounded text-center"
                                            style="background-image: url({{avatar($moderator->user)}})"></a><br>{{$moderator->user->username}}
                                    </div>
                                </td>
                                <td>
                                    <button x-on:click="rm('{{$moderator->id}}')"
                                            class="btn btn-danger btn-sm delete-user">删除
                                    </button>
                                </td>
                            </tr>
                        @endforeach
                        </tbody>
                    </table>
                </div>

            </div>
        </div>
    </div>
    <style>
        /* 移动端样式 */
        @media screen and (max-width: 767px) {
            td {
                word-wrap: break-word;
                overflow: hidden;
                text-overflow: ellipsis;
                max-width: 100px;
            }
        }

    </style>
@endsection

@section('scripts')
    <script>
        document.addEventListener('alpine:init', () => {
            Alpine.data('moderator', () => ({
                rm(id) {
                    swal({
                        title: "确定要删除此用户的版主身份吗？",
                        icon: "warning",
                        buttons: true,
                        dangerMode: true,
                    })
                        .then((willDelete) => {
                            if (willDelete) {
                                const params = {
                                    _token:csrf_token
                                };

                                const url = '/admin/users/moderator/'+id;

                                fetch(url + '?' + new URLSearchParams(params), {
                                    method: 'DELETE',
                                    headers: {
                                        'Content-Type': 'application/json'
                                    }
                                })
                                    .then(response => response.json())
                                    .then(data => {
                                        console.log(data)
                                        if (data.success === true) {
                                            swal('Success', data.result.message, 'success')
                                                .then(() => {
                                                    location.reload();
                                                });
                                        } else {
                                            swal('Error', data.result.message, 'error')
                                        }
                                    })
                            }
                        });
                }
            }))
        })
    </script>
@endsection