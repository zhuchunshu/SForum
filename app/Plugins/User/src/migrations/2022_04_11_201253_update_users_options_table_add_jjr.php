<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateUsersOptionsTableAddJjr extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('users_options', function (Blueprint $table) {
            $table->integer('credits')->default(0)->comment('积分');
			$table->integer('golds')->default(0)->comment('金币');
			$table->integer('exp')->default(0)->comment('经验');
			$table->integer('money')->default(0)->comment('余额');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_options', function (Blueprint $table) {
            //
        });
    }
}
