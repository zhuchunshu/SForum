<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class RmUsersNoticedTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::dropIfExists('users_noticed');
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_noticed', function (Blueprint $table) {
            //
        });
    }
}
