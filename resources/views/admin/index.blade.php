@extends('app')

@section('title','Admin')

@section('content')
    <div class="row row-cards">
        {{--        统计--}}
        <div class="col-md-12">
            <div class="row row-cards">
                {{--                用户总数--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-primary-lt text-white avatar">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24"
     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
     stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>
                                    </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">用户总数</div>
                                    <div class="text-muted">{{\App\Plugins\User\src\Models\User::query()->count()}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>
                {{--                帖子总数--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-red-lt text-white avatar">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-bookmarks" width="24" height="24"
     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
     stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M13 7a2 2 0 0 1 2 2v12l-5 -3l-5 3v-12a2 2 0 0 1 2 -2h6z"></path>
   <path d="M9.265 4a2 2 0 0 1 1.735 -1h6a2 2 0 0 1 2 2v12l-1 -.6"></path>
</svg>
                                    </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">帖子总数</div>
                                    <div class="text-muted">{{\App\Plugins\Topic\src\Models\Topic::query()->count()}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                评论总数--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-pink-lt text-white avatar">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message-circle" width="24" height="24"
     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
     stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
   <line x1="12" y1="12" x2="12" y2="12.01"></line>
   <line x1="8" y1="12" x2="8" y2="12.01"></line>
   <line x1="16" y1="12" x2="16" y2="12.01"></line>
</svg>
                                    </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">评论总数</div>
                                    <div class="text-muted">{{\App\Plugins\Comment\src\Model\TopicComment::query()->count()}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                标签总数--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-green-lt text-white avatar">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-tags" width="24" height="24"
     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
     stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M7.859 6h-2.834a2.025 2.025 0 0 0 -2.025 2.025v2.834c0 .537 .213 1.052 .593 1.432l6.116 6.116a2.025 2.025 0 0 0 2.864 0l2.834 -2.834a2.025 2.025 0 0 0 0 -2.864l-6.117 -6.116a2.025 2.025 0 0 0 -1.431 -.593z"></path>
   <path d="M17.573 18.407l2.834 -2.834a2.025 2.025 0 0 0 0 -2.864l-7.117 -7.116"></path>
   <path d="M6 9h-.01"></path>
</svg>
                                    </span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">标签总数</div>
                                    <div class="text-muted">{{\App\Plugins\Topic\src\Models\TopicTag::query()->count()}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                Swoole版本--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-teal-lt text-white avatar">S</span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">Swoole版本</div>
                                    <div class="text-muted">{{SWOOLE_VERSION}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                PHP版本--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-azure-lt text-white avatar">P</span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">PHP版本</div>
                                    <div class="text-muted">{{PHP_VERSION}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                SuperForum版本--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-indigo-lt text-white avatar">S</span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">系统版本</div>
                                    <div class="text-muted">{{build_info()->version}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>

                {{--                插件数量--}}
                <div class="col-md-6 col-xl-3 col-6">
                    <a class="card card-link" href="#">
                        <div class="card-body">
                            <div class="row">
                                <div class="col-auto">
                                    <span class="bg-yellow-lt text-white avatar"><svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-box" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="12 3 20 7.5 20 16.5 12 21 4 16.5 4 7.5 12 3"></polyline>
   <line x1="12" y1="12" x2="20" y2="7.5"></line>
   <line x1="12" y1="12" x2="12" y2="21"></line>
   <line x1="12" y1="12" x2="4" y2="7.5"></line>
</svg></span>
                                </div>
                                <div class="col">
                                    <div class="font-weight-medium">插件数量</div>
                                    <div class="text-muted">{{count(plugins()->GetAll())}}</div>
                                </div>
                            </div>
                        </div>
                    </a>
                </div>
            </div>
        </div>

        {{--        更新日志--}}
        <div class="col-md-12" id="vue-admin-index-releases">
            <div class="row row-cards">
                <div class="col-md-12" v-if="data">
                    <div class="row row-cards">

                        {{--                        当前版本--}}
                        <div class="col-md-6">
                            <div class="border-0 card" v-if="data">
                                <div class="card-body">
                                    <h3 class="card-title">Releases</h3>
                                    <p>当前版本:@{{data.version}}</p>
                                    <p>最新版本: <a :href="data.new_version_url">@{{data.tag_name}}</a></p>
                                    <div v-if="data.upgrade">
                                        可更新
                                        <br>
                                        <a :href="data.zipball_url" class="btn btn-dark"
                                           style="margin-right: 10px">下载zip包</a>
                                        <a :href="data.tarball_url" style="margin-right: 10px"
                                           class="btn btn-light">下载tar.gz包</a>
                                        <button @@click="update" class="btn btn-green">立即更新</button>
                                    </div>
                                </div>
                                <div class="card-footer">
                                    <div class="col-md-12">
                                        <button @@click="clearCache" class="btn btn-dark">清理缓存</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="border-0 card" v-if="data">
                                <div class="card-body">
                                    <h3 class="card-title">其他信息</h3>
                                    <p>官网: <a href="https://www.runpod.cn">https://www.runpod.cn</a></p>
                                    <p>文档: <a href="https://www.runpod.cn/docs">https://www.runpod.cn/docs</a>
                                    </p>
                                    <p>开源地址: <a href="https://github.com/zhuchunshu/super-forum">https://github.com/zhuchunshu/super-forum</a>
                                    </p>
                                </div>
                            </div>
                        </div>

                        {{--                        commit --}}
                        <div class="col-md-12">
                            <div class="border-0 card" v-if="markdown">
                                <div class="card-body">
                                    <h3 class="card-title">
                                        <div class="row">
                                            <div class="col">更新日志</div>
                                            <div class="col-auto"><a
                                                        href="https://forum.runpod.cn/48.html">查看</a></div>
                                        </div>
                                    </h3>
                                    <div style="overflow:scroll;overflow-x:hidden;max-height: 700px;"
                                         class="markdown" v-html="updateLog"></div>
                                </div>
                            </div>
                        </div>


                    </div>
                </div>

                {{--                加载中--}}
                <div class="col-md-12" v-else>
                    <div class="border-0 card">
                        <div class="card-body">
                            <div class="empty">
                                <p class="empty-title">Loading...</p>

                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix('js/admin/index.js')}}"></script>
@endsection