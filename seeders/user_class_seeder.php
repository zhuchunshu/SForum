<?php

declare(strict_types=1);

use Hyperf\Database\Seeders\Seeder;

class UserClassSeeder extends Seeder
{
    /**
     * Run the database seeds.
     *
     * @return void
     */
    public function run()
    {
        //
        \Hyperf\DbConnection\Db::table('user_class')->insert([
           'name' => '默认用户组',
            'color' => '#206bc4',
            'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>',
            'quanxian' => '["comment_caina","comment_create","comment_edit","comment_remove","report_comment","report_topic","topic_create","topic_delete","topic_edit","topic_tag_create"]',
            'permission-value' => '1',
            'created_at' => date("Y-m-d H:i:s"),
            'updated_at' => date("Y-m-d H:i:s")
        ]);
    }
}
